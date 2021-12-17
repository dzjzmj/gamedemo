package internal

import (
	"fish_server_2021/common"
	"fish_server_2021/game/base"
	"fish_server_2021/game/data"
	"fish_server_2021/game/data/model"
	"lolGF/log"
	"math"
)

// 发子弹
func (room *Room) shoot(player *Player, resp *ShootWrap) {
	result := player.AddOrder(resp.ID)
	if result {
		resp.UID = player.BaseInfo.ID
		resp.Coin = player.BaseInfo.Coin
		player.room.notifyAll(SMShoot, resp.encode())
	} else {
		ret := ShootRet{
			ID:  resp.ID,
			Ret: RetFail,
		}
		base.SendMsgToGate(player.BaseInfo.ID, SMShoot, ret)
		player.SyncCoin()
	}
}
func (room *Room) deadAffect(player *Player, resp *DeadAffectWrap) {
	dead := player.GetSpecialDead(resp.ID)
	var deadMonster map[int64]int64
	if dead != nil {
		deadMonster = room.affectMonster(player, dead, resp)
	}
	//log.Debug("deadMonster %+v %+v", dead, deadMonster)
	retHit := DeadAffectWrap{
		DeadId: resp.DeadId,
		UID:    player.BaseInfo.ID,
		ID:     resp.ID,
		Coin:   player.BaseInfo.Coin,
		Num:    0,
		MID:    make([]int64, 0, 100),
		Win:    make([]int64, 0, 100),
	}
	for mid, win := range deadMonster {
		retHit.Num++
		retHit.MID = append(retHit.MID, mid)
		retHit.Win = append(retHit.Win, win)
	}

	room.notifyAll(SMDeadAffect, retHit.encode())
}

// 影响死亡
func (room *Room) affectMonster(player *Player, dead *PlayerSpecialDead, resp *DeadAffectWrap) map[int64]int64 {
	deadMonster := make(map[int64]int64)
	rateIndex := dead.rateIndex
	rate := player.bulletRates[rateIndex]
	gunRate := rate.Gun
	monsterIds := resp.MID
	//log.Debug("rateIndex %d", rateIndex)

	var allAddCoin int64
	isBlackHole := false
	isLightning := false
	deadData, ok := data.GetMonsterConfig(dead.monster.FormationConfig.Monster)
	if !ok {
		return deadMonster
	}

	if deadData.Type == data.MonsterTypeBlackHole {
		// 黑洞
		isBlackHole = true
	}
	if deadData.Type == data.MonsterTypeLightning {
		// 闪电
		isLightning = true
	}
	for _, monsterId := range monsterIds {
		if dead.skill != nil {
			if dead.skill.Influence > 0 {
				if dead.num >= dead.skill.Influence {
					break
				}
			}
		}

		log.Debug("monsterId%d", monsterId)
		monster, ok := room.monsters[monsterId]
		if !ok || monster == nil {
			continue
		}
		if monster.status == MonsterStatusDeal {
			log.Debug("不存在或死了 %+v %v", monster, ok)
			// 不存在或死了
			continue
		}

		monsterData, ok := data.GetMonsterConfig(monster.FormationConfig.Monster)

		if !ok {
			log.Debug("GetMonsterConfig Is %v", monster.FormationConfig.Monster)
			continue
		}

		var addCoin = monsterData.Rate * gunRate

		if isBlackHole && monsterData.CanBlackhole != 1 {
			continue
		}
		if isLightning && monsterData.CanLightning != 1 {
			continue
		}
		dead.num++

		// 命中
		room.changePool(dead.monster.monsterData, rate, -addCoin, player)
		allAddCoin += addCoin
		deadMonster[monsterId] = addCoin

		revenue := room.getRevenue(monsterData)
		tax := int64(float64(addCoin) * revenue) // 税
		realCoin := addCoin - tax
		player.addCoin(realCoin, true, addCoin)

		data.CreateGameRecord(&model.GameRecord{
			Uid:    player.BaseInfo.ID,
			ItemId: common.CoinItemId,
			Num:    realCoin,
			Mid:    monsterId,
			MType:  monster.monsterData.Id,
			Power:  gunRate,
		})
		monster.status = MonsterStatusDeal
	}
	if allAddCoin > 0 {
		skeleton.GoSafe(func() {
			common.TaskTriggerAddItem(player.BaseInfo.ID, common.CoinItemId, 1, allAddCoin)
		})
	}

	return deadMonster
}
func (room *Room) hit(player *Player, resp *HitWrap) {
	//log.Info("hit:%v %+v", player, resp)
	rateIndex := player.DecOrderValue(resp.ID)
	if rateIndex < 0 { // 无效子弹
		//log.Debug("无效子弹 %d", resp.ID)
		return
	}
	deadMonster, addItemId := room.hitMonster(player, rateIndex, false, resp)
	log.Debug("deadMonster %+v", deadMonster)
	retHit := HitWrap{
		UID:       player.BaseInfo.ID,
		ID:        resp.ID,
		Coin:      player.BaseInfo.Coin,
		Num:       0,
		MID:       make([]int64, 0, 100),
		Win:       make([]int64, 0, 100),
		AddItemId: addItemId,
	}
	for _, d := range deadMonster {
		retHit.Num++
		for mid, win := range d {
			retHit.MID = append(retHit.MID, mid)
			retHit.Win = append(retHit.Win, win)
		}

	}

	if retHit.Num > 0 {
		room.notifyAll(SMHit, retHit.encode())
	}
}

// 攻击
func (room *Room) hitMonster(player *Player, rateIndex int, ignoreProb bool, resp *HitWrap) (deadMonster []map[int64]int64, addItemId int) {
	deadMonster = make([]map[int64]int64, 0, 20)
	rate := player.bulletRates[rateIndex]
	gunRate := rate.Gun
	monsterIds := resp.MID
	//log.Debug("rateIndex %d", rateIndex)

	var allAddCoin int64
	isFirst := true
	isBlackHole := false
	isLightning := false
	isBomb := false
	isBoss := false
	isBombHit := false
	skeleton.GoSafe(func() {
		player.addRadPackBullet(rate)
	})
	addItemId = common.CoinItemId
	var boss *Monster

	isRefund := true
	for _, monsterId := range monsterIds {

		//log.Debug("monsterId %d", monsterId)
		room.configMonsterNumLock.Lock()
		monster, ok := room.monsters[monsterId]
		room.configMonsterNumLock.Unlock()
		if !ok || monster == nil {
			base.SendMsgToGate(player.BaseInfo.ID, SMReadPackBulletNum,
				RedPackBulletNum{BulletNum: player.BaseInfo.BulletNum})
			log.Debug("不存在 %+v %v", monster, ok)
			continue
		}
		if monster.status == MonsterStatusDeal {
			base.SendMsgToGate(player.BaseInfo.ID, SMReadPackBulletNum,
				RedPackBulletNum{BulletNum: player.BaseInfo.BulletNum})
			log.Debug("死了 %+v %v", monster, ok)
			// 不存在或死了
			continue
		}
		skeleton.GoSafe(func() {
			player.bulletBuff(monster)
		})

		//monsterData, ok := data.GetMonsterConfig(monster.FormationConfig.Monster)
		monsterData := monster.monsterData
		if !ok {
			log.Debug("GetMonsterConfig Is %v", monster.FormationConfig.Monster)
			continue
		}
		if isFirst {
			room.changePool(monsterData, rate, gunRate, player)
			data.CreateGameRecord(&model.GameRecord{
				Uid:    player.BaseInfo.ID,
				ItemId: common.CoinItemId,
				Num:    -gunRate,
				Mid:    monsterId,
				MType:  monster.monsterData.Id,
				Power:  gunRate,
			})
		}
		if monsterData.Type == data.MonsterTypeBlackHole {
			// 黑洞
			isBlackHole = true
		}
		if monsterData.Type == data.MonsterTypeLightning {
			// 闪电
			isLightning = true
		}
		if monsterData.Type == data.MonsterTypeBomb {
			// 炸弹
			isBomb = true
		}
		if monsterData.CloseType == data.CloseTypeBoss {
			isBoss = true
		}

		isFirst = false
		isRefund = false
		if !ignoreProb {
			if !room.hitResult(player, rate, monsterData, monster) {
				continue
			}
		}
		if isBlackHole || isLightning {
			// 特殊怪技能保存
			player.AddSpecialDead(resp.ID, monster, rateIndex)

			//if isBlackHole && !ignoreProb && monsterData.Rate > 15 {
			//	continue
			//}
			//if isLightning && !ignoreProb && monsterData.Rate > 20 {
			//	continue
			//}
			//ignoreProb = true

		}
		if isBomb || isBoss {
			isBombHit = true
			boss = monster
			//goto isBombLabel
		}
		// 命中
		var addCoin = monsterData.Rate * gunRate
		if monsterData.CloseType == data.CloseTypeRedPack {
			addCoin = int64(float32(monsterData.Rate*gunRate) * monsterData.Redpack)
			addItemId = common.PearlItemId
		}
		allAddCoin += addCoin
		//deadMonster[monsterId] = addCoin
		deadMonster = append(deadMonster, map[int64]int64{monsterId: addCoin})

		revenue := room.getRevenue(monsterData)
		tax := int64(math.Ceil(float64(addCoin-gunRate) * revenue / 100)) // 税
		realCoin := addCoin - tax
		//log.Debug("revenue %v addCoin%v tax%v realCoin%v", revenue, addCoin, tax, realCoin)
		if addItemId == common.CoinItemId {
			player.addCoin(realCoin, true, addCoin)
			data.CreateGameRecord(&model.GameRecord{
				Uid:    player.BaseInfo.ID,
				ItemId: common.CoinItemId,
				Num:    realCoin,
				Mid:    monsterId,
				MType:  monster.monsterData.Id,
				Power:  gunRate,
			})
		} else if addItemId == common.PearlItemId {
			player.addPearl(realCoin, 1, "", false)
			data.CreateGameRecord(&model.GameRecord{
				Uid:    player.BaseInfo.ID,
				ItemId: common.PearlItemId,
				Num:    realCoin,
				Mid:    monsterId,
				MType:  monster.monsterData.Id,
				Power:  gunRate,
			})
		}
		skeleton.GoSafe(func() {
			common.TaskTriggerKillMonster(player.BaseInfo.ID, monsterData.Id, monsterData.Rate, monsterData.Type, 1)
			room.drop(player, monsterData, monsterId)
			room.changePool(monsterData, rate, -realCoin, player)
			if room.RoomType.RoomType == data.RoomTypeCasino {
				room.getCasinoCoin(player, monsterData, rate, monsterId)
			}
		})
		monster.status = MonsterStatusDeal
	}
	if isRefund {
		// 没打到
		player.refundCoin(gunRate, false)
	}
	//isBombLabel:
	if isBombHit {
		for monsterId, monster := range room.monsters {
			monsterData, ok := data.GetMonsterConfig(monster.FormationConfig.Monster)

			if !ok {
				log.Debug("GetMonsterConfig Is %v", monster.FormationConfig.Monster)
				continue
			}
			var addCoin = monsterData.Rate * gunRate
			if (isBoss && monsterData.CanBoss == 1) || (isBomb && monsterData.CanBomb == 1) {
				if boss != nil {
					room.changePool(boss.monsterData, rate, -addCoin, player)
				}
				allAddCoin += addCoin
				//deadMonster[monsterId] = addCoin
				deadMonster = append(deadMonster, map[int64]int64{monsterId: addCoin})
				monster.status = MonsterStatusDeal

				revenue := room.getRevenue(monsterData)
				tax := int64(float64(addCoin) * revenue) // 税
				realCoin := addCoin - tax
				player.addCoin(realCoin, true, addCoin)
				data.CreateGameRecord(&model.GameRecord{
					Uid:    player.BaseInfo.ID,
					ItemId: common.CoinItemId,
					Num:    realCoin,
					Mid:    monsterId,
					MType:  monster.monsterData.Id,
					Power:  gunRate,
				})
			}
		}
	}
	if allAddCoin > 0 {
		skeleton.GoSafe(func() {
			common.TaskTriggerAddItem(player.BaseInfo.ID, 1, 1, allAddCoin)
		})
	}

	return deadMonster, addItemId
}

var totalWeight int32 = 1000000

// 通过概率计算死亡
func (room *Room) hitResult(player *Player, bulletRate *data.BulletRate, monster *data.Monster, m *Monster) bool {
	poolCoefficient := room.getPoolCoefficient(monster, bulletRate)
	if poolCoefficient == -1 {
		//log.Debug("poolCoefficient %v", poolCoefficient)
		return false
	}
	weight := monster.Weight
	monsterWeight, playerWeight := player.newPlayerPoolCoefficient(monster, bulletRate)
	if monsterWeight >= 0 {
		weight = monsterWeight
	}
	//log.Debug("newPlayerPoolCoefficient %d %d %d", weight, monsterWeight, playerWeight)
	if weight <= 0 {
		return false
	}
	rateByMonster := bulletRateByMonster(bulletRate, monster)
	// 怪物爆率 * (100000 + 子弹微调系数 + 水池微调系数 + 新手保护系数) / 100000
	prob := int32(float64(weight) * (float64(totalWeight+rateByMonster+poolCoefficient+playerWeight) / float64(totalWeight))) //* bulletRate.Gun
	//log.Debug("hitResult %d,%d,%d,%d,%d", weight, rateByMonster, poolCoefficient, playerWeight, monster.Id)
	randTmp := randSource.Int31n(totalWeight) + 1
	hitResult := randTmp < prob
	//log.Debug("hitResult%v %d %d %d %d", hitResult, totalWeight, randTmp, prob, monster.Id)

	// 测试用 上线删除
	//tmp := map[string]interface{}{
	//	"怪物爆率":      weight,
	//	"爆率":        prob,
	//	"子弹微调系数":    rateByMonster,
	//	"水池微调系数":    poolCoefficient,
	//	"个人系数":      playerWeight,
	//	"rand":      randTmp,
	//	"hitResult": hitResult,
	//	"MID":       m.ID,
	//	"MType":     monster.Id,
	//}
	//base.SendMsgToGate(player.BaseInfo.ID, 9903, tmp)

	return hitResult
}

var LimitSkillDay int64 = 100000

func (player *Player) checkDropLimit(itemId int32) bool {
	if itemId > 0 {
		item := data.GetItemConfig(itemId)
		if player.lastDropInDay != currTimeInDay {
			player.dropItemNumLock.Lock()
			player.dropItemNum[itemId] = 0
			player.dropItemNum[-item.Type] = 0
			player.dropItemNumLock.Unlock()
			player.lastDropInDay = currTimeInDay

		} else {
			player.dropItemNumLock.RLock()
			num := player.dropItemNum[itemId]
			typeNum := player.dropItemNum[-item.Type]
			player.dropItemNumLock.RUnlock()
			limit := common.GetDropItemLimit(itemId)
			if (limit > 0 && num >= limit) || limit == -1 {
				r := randSource.Intn(500)
				if r > 0 {
					return true
				}
			}

			limitType := common.GetDropItemLimit(-item.Type)
			if (limitType > 0 && typeNum >= limitType) || limitType == -1 {
				r := randSource.Intn(500)
				if r > 0 {
					return true
				}
			}
		}
	} else {
		if player.lastDropInDay != currTimeInDay {
			player.dropNum = 0
			player.lastDropInDay = currTimeInDay
		} else {

			if player.dropNum >= LimitSkillDay {
				r := randSource.Intn(500)
				if r > 0 {
					return true
				}
			}
		}
	}
	return false
}
func (room *Room) getCasinoCoin(player *Player, monsterData *data.Monster, rate *data.BulletRate, mid int64) {
	c := monsterData.CasinoScore * rate.Gun / 100
	if c > 0 {
		player.addItem(common.CasinoCoinItemId, c, mid, 1, "")
	}
}
func (room *Room) drop(player *Player, monsterData *data.Monster, mid int64) {
	if monsterData.DropID == 0 {
		return
	}
	if player.checkDropLimit(0) {
		return
	}
	drops, ok := data.GetDropGroupConfigs(monsterData.DropID)
	if ok {
		weights := make([]int, len(drops))
		for i, drop := range drops {
			weights[i] = drop.Weight
		}
		index := weightRandomIndex(slice2map(weights))
		drop := drops[index]
		if drop.Item == 0 {
			return
		}
		if player.checkDropLimit(drop.Item) {
			return
		}
		player.dropNum += drop.Number // 总数
		item := data.GetItemConfig(drop.Item)

		player.dropItemNumLock.Lock()
		player.dropItemNum[drop.Item] += drop.Number  // 物品数
		player.dropItemNum[-item.Type] += drop.Number // 物品分类数
		player.dropItemNumLock.Unlock()
		// 掉落
		player.addItem(drop.Item, drop.Number, mid, 1, "")

	}

}

func bulletRateByMonster(rate *data.BulletRate, monster *data.Monster) int32 {
	var ret int32
	if monster.Type == data.MonsterTypeSmall {
		ret = rate.MonsterRate
	} else if monster.Type == data.MonsterTypeRedPack {
		ret = rate.RedpackRate
	} else if monster.Type == data.MonsterTypeLightning ||
		monster.Type == data.MonsterTypeBlackHole || monster.Type == data.MonsterTypeBomb {
		ret = rate.SpecialRate
	} else if monster.Type == data.MonsterTypeBoss {
		ret = rate.BoosRate
	} else if monster.Type == data.MonsterTypePool {
		ret = rate.ChestRate
	}
	return ret
}
func (player *Player) decPowerToUsable() {
StartLabel:
	player.currentBulletRate--
	rate := player.bulletRates[player.currentBulletRate]
	gun := rate.Gun
	if player.currentBulletRate > 0 && gun > player.BaseInfo.Coin {
		goto StartLabel
	}

	player.notifyPower()
}
func (player *Player) changePower(req *ChangePowerReq) {
	bulletLen := len(player.bulletRates)
	//log.Debug("start %v %v", player.currentBulletRate, bulletLen)
	if req.CT == ChangePowerDec {
		if player.currentBulletRate == 0 {
			player.currentBulletRate = bulletLen - 1
		} else {
			player.currentBulletRate--
		}
	} else if req.CT == ChangePowerAdd {
		if player.currentBulletRate >= bulletLen-1 {
			player.currentBulletRate = 0
		} else {
			//log.Debug("add-----")
			player.currentBulletRate++
		}
	} else if req.CT == ChangePowerTop {
		player.currentBulletRate = bulletLen - 1
	}

	player.notifyPower()
}

func (player *Player) notifyPower() {
	//log.Debug("end %v", player.currentBulletRate)
	rate := player.bulletRates[player.currentBulletRate]
	//log.Debug("rate %+v", rate)
	ret := ChangePowerRet{
		UID:   player.BaseInfo.ID,
		Ret:   1,
		Power: rate.Gun,
	}
	player.BaseInfo.Power = rate.Gun
	player.room.notifyAll(SMChangePower, ret)
	//base.SendMsgToGate(player.BaseInfo.ID, )
}

// 税收
func (room *Room) getRevenue(monster *data.Monster) float64 {
	var revenue float64
	if monster.CloseType == data.CloseTypeSmall {
		revenue = room.RoomType.MonsterRevenue
	} else if monster.CloseType == data.CloseTypeRedPack {
		revenue = room.RoomType.RedpackRevenue
	} else if monster.CloseType == data.CloseTypeSpecial {
		revenue = room.RoomType.SpecialRevenue
	} else if monster.CloseType == data.CloseTypeBoss {
		revenue = room.RoomType.BossRevenue
	} else if monster.CloseType == data.CloseTypePool {
		revenue = room.RoomType.ChestRevenue
	}
	return revenue
}
