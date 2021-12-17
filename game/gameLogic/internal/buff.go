package internal

import (
	"fish_server_2021/game/base"
	"fish_server_2021/game/data"
	"lolGF/log"
	"lolGF/utils/econv"
	"math/rand"
	"time"
)

func (player *Player) bulletBuff(monster *Monster) {
	if player.Hero == nil {
		return
	}
	if player.Hero.Bullet == nil {
		return
	}
	if player.Hero.Bullet.BuffID == 0 || player.Hero.Bullet.Probability == 0 {
		return
	}

	r := randSource.Intn(101) + 1
	if r <= player.Hero.Bullet.Probability {
		// 触发buff
		buff, ok := data.GetBuffConfig(player.Hero.Bullet.BuffID)
		if ok {
			player.room.notifyAll(SMBuff, []interface{}{monster.ID, player.Hero.Bullet.BuffID})

			if buff.Type == 3 { //原地不动
				monster.buffTime += buff.Duration
			} else if buff.Type == 1 { // 减速
				monster.buffTime += buff.Duration * buff.TypeData / 100
			} else if buff.Type == 2 { // 加速
				monster.buffTime -= buff.Duration * buff.TypeData / 100
			}
		}
	}

}

func (room *Room) monsterSkill(delay int, newMonster *Monster, skill int) {
	skeleton.AfterFunc(time.Duration(delay)*time.Second, func() {
		skillConfig, ok := data.GetSkillConfig(skill)
		if ok {
			monsterIds := make([]int64, 0, 20)
			if skillConfig.Type == 6 {
				// 恐吓逃跑效果：对全屏的小怪，有%20的概率产生恐吓，被恐吓到的小怪，增加50%的移动速度，持续5秒
				room.configMonsterNumLock.RLock()
				for id, m := range room.monsters {
					if m.monsterData.Type == 1 { // 小怪
						r := rand.Intn(100)
						if r <= skillConfig.Probability {
							// 触发buff
							monsterIds = append(monsterIds, id)
						}
					}
				}
				room.configMonsterNumLock.RUnlock()

			} else if skillConfig.Type == 5 {
				// 落石眩晕效果：随机对5只小怪造成眩晕效果，持续5秒，眩晕时无法移动
				n := skillConfig.Influence
				room.configMonsterNumLock.RLock()
				minMonster := make([]int64, 0, 100)
				for id, m := range room.monsters {
					if m.monsterData.Type == 1 { // 小怪
						minMonster = append(minMonster, id)
					}
				}
				minMonsterLen := len(minMonster)
				if minMonsterLen > 5 {
					// 打乱
					for k := range minMonster {
						r := rand.Intn(minMonsterLen)
						if r != k {
							minMonster[k], minMonster[r] = minMonster[r], minMonster[k]
						}

					}
				} else {
					n = minMonsterLen
				}
				for i := 0; i < n; i++ {
					monsterIds = append(monsterIds, minMonster[i])
				}
				room.configMonsterNumLock.RUnlock()
			}
			room.notifyAll(SMSkill, []interface{}{newMonster.ID, skill, skillConfig.BuffId, monsterIds})
		}
		// 产生技能

	})
}

func (player *Player) skillSwitch(req *SkillSwitchReq) {
	ret, ok := player.skillConfig[req.Skill]
	if !ok {
		log.Debug("no data %+v", req)
		return
	}
	ret.Skill = req.Skill
	if ret.isOpen == true {
		ret.Ret = 3
		base.SendMsgToGate(player.BaseInfo.ID, SMSKillSwitch, ret)
		return
	}
	// 冷却时间未到
	if ret.isCool == true {
		ret.Ret = 4
		base.SendMsgToGate(player.BaseInfo.ID, SMSKillSwitch, ret)
		return
	}
	skill := data.GetSkill(ret.Skill, player.BaseInfo.Hero)
	str, _ := econv.String(ret.Skill)
	success := false
	if len(skill.ConsumeItems) > 0 {
		for _, consume := range skill.ConsumeItems {
			ok := player.subItem(consume.Id, consume.Num, 0, 4, str)

			if ok {
				success = true
				break
			}
		}
	} else {
		success = true
	}

	if !success {
		ret.Ret = 5
		base.SendMsgToGate(player.BaseInfo.ID, SMSKillSwitch, ret)
		return
	}
	currentBulletHP = skillBulletHP
	currMillisecond := time.Now().UnixNano() / 1000 / 1000
	ret.isOpen = true
	ret.isCool = true
	ret.UID = player.BaseInfo.ID
	ret.Ret = 1
	ret.Start = currMillisecond
	ret.Used = 0

	if skill != nil {
		ret.Sec = skill.SkillTime
		ret.CD = skill.CD
	}

	player.room.notifyAll(SMSKillSwitch, ret)

	skeleton.AfterFunc(time.Second*time.Duration(ret.Sec), func() {
		currentBulletHP = 1
		ret.isOpen = false
	})
	skeleton.AfterFunc(time.Second*time.Duration(ret.CD), func() {
		ret.isCool = false
		ret.Start = 0
	})

}
