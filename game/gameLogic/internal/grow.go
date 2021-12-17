package internal

import (
	"fish_server_2021/game/data"
	"lolGF/module"
	"math"
	"math/rand"
	"sync/atomic"
	"time"
)

// 刷怪规则
type MonsterGrowConfig struct {
	Config          *data.MonsterGrowConfig
	Timer           *module.Timer
	Doing           bool
	PerMonsterIndex []int //上次生成的怪
}

func (room *Room) newMonsterId() int64 {
	return atomic.AddInt64(&room.monsterCurrId, 1)
}

func (room *Room) growMonsters() {

	for _, config := range room.MonsterGrow {
		config.PerMonsterIndex = make([]int, 0, 10)
		c := config
		if config.Config.Appear == 0 {
			skeleton.GoSafe(func() {
				room.growMonster(c, true)
			})
		} else {
			skeleton.AfterFunc(time.Duration(c.Config.Appear)*time.Second, func() {
				skeleton.GoSafe(func() {
					room.growMonster(c, true)
				})
			})
		}

	}
	// 大军
	if room.Map != nil && room.Map.InvadeTime > 0 {
		room.ArmyTimer = skeleton.AfterFunc(time.Duration(room.Map.InvadeTime-6)*time.Second, func() {
			skeleton.GoSafe(func() {
				room.growHugeArmyNotice()
			})
		})
	}

}

// 大军来袭提前通知
func (room *Room) growHugeArmyNotice() {
	for _, config := range room.MonsterGrow {
		if config.Timer != nil {
			config.Timer.Stop()
		}
	}
	room.isGrowHugeArmy = true

	// 提前前通知
	// 切换地图
	oldMap := room.Map
	newMap := room.changeMap()
	ret := HugeArmyRet{NextMap: newMap.Id, Type: 1}
	room.notifyAll(SMHugeArmy, ret)

	skeleton.AfterFunc(6*time.Second, func() {
		room.monsters = make(map[int64]*Monster)
		room.configMonsterNum = make(map[int]int32)
		ret := HugeArmyRet{NextMap: newMap.Id, Type: 2}
		room.notifyAll(SMHugeArmy, ret)
		skeleton.GoSafe(func() {
			room.growHugeArmy(oldMap)
		})
	})
}

// 大军来袭
func (room *Room) growHugeArmy(p *data.Map) {
	weightIndex := weightRandomIndex(slice2map(p.Weight))
	if len(p.InvadeGroup) >= weightIndex+1 {
		monsterGroup := p.InvadeGroup[weightIndex]
		formations := data.GetFormationConfigs(monsterGroup)
		for _, formation := range formations {
			if formation.Delay > 0 {
				f := formation
				skeleton.AfterFunc(time.Duration(formation.Delay*1000)*time.Millisecond, func() {
					room.newMonster(f, nil)
				})
			} else {
				room.newMonster(formation, nil)
			}

		}
	}
	skeleton.AfterFunc(time.Duration(room.Map.IntervalTime[weightIndex])*time.Second, func() {
		room.isGrowHugeArmy = false
		room.growMonsters()
	})

}

func (room *Room) growMonster(config *MonsterGrowConfig, isFirst bool) {
	var weightIndex int
	if room.isGrowHugeArmy {
		// 大军来时暂停
		return
	}
	if !isFirst {
		room.configMonsterNumLock.RLock()
		// 判断数量是否够
		n := room.configMonsterNum[config.Config.Id]
		room.configMonsterNumLock.RUnlock()
		if n >= config.Config.MaxNum {
			goto nextLoop
		}
	}
	config.Doing = true
	if config.Config.Type == data.MonsterFormationTypeOne { //单怪
		growNums := 0
		//preIndex := -1
		//room.perMonsterIndexLock.Lock()
		perMonsterIndex := config.PerMonsterIndex
		config.PerMonsterIndex = make([]int, 0, 10)
		//room.perMonsterIndexLock.Unlock()
		for {
			if room.isGrowHugeArmy {
				goto nextLoop
			}
			if len(config.Config.Monster) > 1 {
				newWeight := make(map[int]int)
				for i, w := range config.Config.Weight {
					if !inArray(i, perMonsterIndex) {
						newWeight[i] = w
					}
				}
				if len(newWeight) <= 1 {
					for i := range newWeight {
						weightIndex = i
					}
					perMonsterIndex = make([]int, 0, 10)

				} else {
					weightIndex = weightRandomIndex(newWeight)

					perMonsterIndex = append(perMonsterIndex, weightIndex)

				}
			} else {
				weightIndex = 0
			}
			if len(config.Config.Monster) >= weightIndex+1 {
				monsterGroup := config.Config.Monster[weightIndex]
				formations := data.GetFormationConfigs(monsterGroup)
				formationLen := len(formations)
				if formationLen > 0 {
					var n int
					if formationLen == 1 {
						n = 0
					} else {
						n = rand.Intn(formationLen)
					}

					formation := formations[n]
					//fmt.Printf("单怪 %d \n", n)
					ret := room.newMonster(formation, config)
					if ret == false {
						goto nextLoop
					}
				}
			}
			//preIndex = weightIndex
			growNums++
			if growNums >= config.Config.Num {
				break
			}
			if config.Config.Interval > 0 {
				time.Sleep(time.Duration(config.Config.Interval) * time.Second)
			}
		}
		//room.perMonsterIndexLock.Lock()
		config.PerMonsterIndex = perMonsterIndex
		//room.perMonsterIndexLock.Unlock()

	} else {
		//rand.Seed(time.Now().Unix())
		weightIndex = weightRandomIndex(slice2map(config.Config.Weight))
		if len(config.Config.Monster) >= weightIndex+1 {

			monsterGroup := config.Config.Monster[weightIndex]
			formations := data.GetFormationConfigs(monsterGroup)
			formationLen := len(formations)
			if formationLen > 0 {
				for i := 0; i < config.Config.Num; i++ {
					for _, formation := range formations {
						if room.isGrowHugeArmy {
							goto nextLoop
						}
						if formation.Delay > 0 {
							//time.Sleep(time.Duration(formation.Delay) * time.Second)
							f := formation
							c := config
							skeleton.AfterFunc(time.Duration(formation.Delay*1000)*time.Millisecond, func() {
								if room.isGrowHugeArmy {
									return
								}
								room.newMonster(f, c)
							})
						} else {
							room.newMonster(formation, config)

						}
						//fmt.Println("陈型")
					}
				}

			}
		}
	}
	config.Doing = false
nextLoop:
	if config.Config.Loop > 0 {
		config.Timer = skeleton.AfterFunc(time.Duration(config.Config.Loop)*time.Second, func() {
			skeleton.GoSafe(func() {
				room.growMonster(config, false)
			})
		})
	}
}

func (room *Room) newMonster(formation *data.MonsterFormationConfig, config *MonsterGrowConfig) bool {
	room.configMonsterNumLock.Lock()
	defer room.configMonsterNumLock.Unlock()
	pathConfig, ok := data.GetMonsterPathConfig(formation.Path)
	live := formation.Live
	if ok && live <= 0 {
		live = int64(math.Ceil(pathConfig.Length / formation.Speed * float64(formation.Traverse)))
	}
	currMillisecond := time.Now().UnixNano() / 1000 / 1000
	newMonster := &Monster{
		ID:              room.newMonsterId(),
		FormationConfig: formation,
		GrowTime:        currMillisecond,
		status:          MonsterStatusLive,
		config:          config,
		Live:            live,
	}
	monsterData, ok := data.GetMonsterConfig(formation.Monster)
	if !ok {
		return true
	}
	newMonster.monsterData = monsterData
	//log.Debug("newMonster %v", newMonster)
	room.monsters[newMonster.ID] = newMonster

	ret := true
	if config != nil {
		room.configMonsterNum[config.Config.Id]++
		ret = !(room.configMonsterNum[config.Config.Id] >= config.Config.MaxNum)
	}
	//fmt.Println(time.Now().Unix(), newMonster.ID, newMonster.FormationConfig.ID)
	// 通知客户端
	room.sendMonsterRet(0, newMonster, false)
	for i, skill := range formation.SkillId {
		t := formation.Time[i]
		if t > 0 {
			room.monsterSkill(t, newMonster, skill)
		}

	}
	return ret
}

func (room *Room) monsterDie() {
	room.configMonsterNumLock.Lock()

	decConfigs := make(map[int]*MonsterGrowConfig)
	currMillisecond := time.Now().UnixNano() / 1000 / 1000
	for id, monster := range room.monsters {
		if (monster.Live > 0 && (currMillisecond-monster.GrowTime)/1000 > monster.Live) ||
			monster.status == MonsterStatusDeal {
			delete(room.monsters, id)
			if monster.config != nil {
				room.configMonsterNum[monster.config.Config.Id]--

				if monster.config.Config.MinNum > 0 &&
					room.configMonsterNum[monster.config.Config.Id] < monster.config.Config.MinNum {
					decConfigs[monster.config.Config.Id] = monster.config
				}
			}
		}
	}
	room.configMonsterNumLock.Unlock()
	// 补充怪的数量
	room.isAddMonster(decConfigs)

}
func (room *Room) isAddMonster(configs map[int]*MonsterGrowConfig) {
	for _, config := range configs {
		if !config.Doing {
			if config.Timer != nil {
				config.Timer.Stop()
			}
			room.growMonster(config, true)
		}
	}
}
