package internal

import (
	"fish_server_2021/common"
	"fish_server_2021/game/base"
	"fish_server_2021/game/data"
)

type AwardItem struct {
	Id       int32 // 物品ID
	Num      int64 // 数量
	Class    int   // 分类
	configId int   // 阶段ID
}
type RedPackAwardConfig struct {
	Id        int   // 当前ID
	AddNumber int64 // 当前总长度
	AccNumber int64 // 累计之前的长度
	BulletNum int64 // 当前累计子弹数
	FinishId  int   // 可领红包阶段ID
}
type RedPackBulletNum struct {
	BulletNum int64 // 当前累计子弹数
}
type LevelConfigRet struct {
	ID    int
	UID   int32
	Award []data.AwardItem
}

func (player *Player) nextLevelConfig() {
	player.nextLevel = -1
	configs := data.AllLevelConfigs()
	for _, l := range configs {
		if l.ID > player.BaseInfo.UserLevel {
			player.nextLevel = l.ID
			player.nextLevelNum = l.AllExp
			break
		}
	}
	//fmt.Printf("%+v %v",configs,player.nextLevel)
}
func (player *Player) updateLevel() {
	// 判断升级
	if player.nextLevel == 0 {
		player.nextLevelConfig()
	}
	if player.nextLevel == -1 { // 最高级
		return
	}
	if player.BaseInfo.Exp >= player.nextLevelNum {
		player.BaseInfo.UserLevel = player.nextLevel
		// 保存
		skeleton.GoSafe(func() {
			common.UpdateUserLevel(player.BaseInfo.ID, player.BaseInfo.UserLevel)
		})
		// 发送消息
		levelConfig := data.GetLevelConfig(player.nextLevel)
		player.room.notifyAll(SMLevelUp, LevelConfigRet{
			ID:    levelConfig.ID,
			UID:   player.BaseInfo.ID,
			Award: levelConfig.Award,
		})
		player.upLevelAward = levelConfig.Award
		player.nextLevelConfig()

	}
}

func (player *Player) getUpLevelAward() {
	if player.upLevelAward == nil {
		return
	}
	for _, award := range player.upLevelAward {
		player.addItem(award.Id, award.Num, -3, 1, "")

	}

	player.upLevelAward = nil
	base.SendRetToGate(player.BaseInfo.ID, SMLevelUpAward, 1)
}

func (player *Player) addRadPackBullet(rate *data.BulletRate) {
	player.BaseInfo.BulletNum += rate.Gun
	player.BaseInfo.Exp += rate.Gun
	player.checkRedPackFinish(false)
	player.updateLevel()

	common.TaskTriggerShoot(player.BaseInfo.ID, rate.Gun)
	common.TaskTriggerUseItem(player.BaseInfo.ID, 1, 1, rate.Gun)
}
func (player *Player) checkRedPackFinish(isRestart bool) {
restartLabel:
	// 数量是否足够
	configs := data.GetRedPackAwardConfigs()
	config := configs[player.currAwardIndex]

	if player.BaseInfo.BulletNum >= config.AccNumber {
		if player.waitGetAward != nil {
			if player.waitGetAward.configId == config.Id {
				return
			}
		}
		// 生成奖励
		i := weightRandomIndex(slice2map(config.Probability))
		//log.Debug("addRadPackBullet %+v %d", config, i)

		awardItem := config.AwardItem[i]

		item := &AwardItem{
			Id:       awardItem.Id,
			Num:      awardItem.Num,
			configId: config.Id,
		}

		player.waitGetAward = item
		//base.SendMsgToGate(player.BaseInfo.ID, SMReadPackAward, config.AwardItem)
		if !isRestart {
			base.SendMsgToGate(player.BaseInfo.ID, SMReadPackAward, RedPackRet{Ret: 1, Id: config.Id})
		}

		// 进入下一阶段
		if player.currAwardIndex+1 < len(configs) {
			player.currAwardIndex++
			if !isRestart {
				player.sendRedPackAwardConfig()
			}
			if isRestart {
				goto restartLabel
			}

		}
	}
}

func (player *Player) getRedPackAward() {
	if player.waitGetAward == nil {
		return
	}
	player.addItem(player.waitGetAward.Id, player.waitGetAward.Num, -1, 1, "")
	base.SendMsgToGate(player.BaseInfo.ID, SMReadPackResult, player.waitGetAward)

	common.TaskTriggerRedPack(player.BaseInfo.ID, player.waitGetAward.configId)

	player.waitGetAward = nil

	player.currAwardIndex = 0

	player.BaseInfo.BulletNum = 0

	player.sendRedPackAwardConfig()
}

func (player *Player) sendRedPackAwardConfig() {
	configs := data.GetRedPackAwardConfigs()
	config := configs[player.currAwardIndex]
	ra := RedPackAwardConfig{
		Id:        config.Id,
		AddNumber: config.AddNumber,
		AccNumber: config.AccNumber,
		BulletNum: player.BaseInfo.BulletNum,
		// 当前时度子弹=BulletNum-(AccNumber-AddNumber)
	}
	if player.waitGetAward != nil {
		ra.FinishId = player.waitGetAward.configId
	}
	base.SendMsgToGate(player.BaseInfo.ID, SMRedPackAwardConfig, ra)
}
