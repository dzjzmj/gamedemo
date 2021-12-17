package internal

import (
	"fish_server_2021/common"
	"fish_server_2021/common/proto"
	"fish_server_2021/game/base"
	"fish_server_2021/game/data"
)

func (player *Player) upPower() {
	// 扣除宝石
	if player.nextFire != nil {
		ok := player.subItem(common.GemItemId, player.nextFire.Gem, 0, 1, "")
		if !ok {
			return
		}
		player.BaseInfo.MaxPower = player.nextFire.Power
		for _, award := range player.nextFire.Award {
			player.addItem(award.Id, award.Num, -2, 1, "")
		}
		// 保存当前火力
		_ = common.ItemClient.SetItem(proto.ItemReq{
			Uid:     player.BaseInfo.ID,
			ItemId:  common.MaxPowerItemId,
			Num:     player.BaseInfo.MaxPower,
			Action:  1,
			Service: "game",
		})
		common.TaskTriggerUpPower(player.BaseInfo.ID, player.BaseInfo.MaxPower)
		// 更新下级火力
		fires := data.GetFirePowerConfigs()
		player.nextFire = nil
		for _, fire := range fires {
			if fire.Power > player.BaseInfo.MaxPower {
				player.nextFire = fire
				break
			}
		}

		// 返回通知
		base.SendRetToGate(player.BaseInfo.ID, SMUpPower, 1)
		player.upPowerNotice(true)
		bulletLen := len(player.bulletRates)
		hasNew := player.setBulletRates()
		if player.currentBulletRate >= bulletLen-1 && hasNew {
			// 切换到最高火力
			player.changePower(&ChangePowerReq{CT: ChangePowerTop})
		}
	}
}
func (player *Player) upPowerNotice(isNotice bool) {
	if player.nextFire != nil {
		if player.BaseInfo.Gem >= player.nextFire.Gem {
			// 通知升级
			ret := UpPowerRet{
				MaxPower: player.BaseInfo.MaxPower,
				Config:   player.nextFire,
				Ok:       true,
			}
			base.SendMsgToGate(player.BaseInfo.ID, SMUpPowerNotice, ret)
		} else {
			if isNotice {
				ret := UpPowerRet{
					MaxPower: player.BaseInfo.MaxPower,
					Config:   player.nextFire,
					Ok:       false,
				}
				base.SendMsgToGate(player.BaseInfo.ID, SMUpPowerNotice, ret)
			}
		}
	} else {
		ret := UpPowerRet{
			MaxPower: player.BaseInfo.MaxPower,
			Config:   nil,
			Ok:       false,
		}
		base.SendMsgToGate(player.BaseInfo.ID, SMUpPowerNotice, ret)
	}
}

func (player *Player) setBulletRates() bool {
	oldLen := 0
	if player.bulletRates != nil {
		oldLen = len(player.bulletRates)
	}
	if player.room != nil {
		room := player.room
		if room.bulletRates != nil {
			//log.Debug("BaseInfo %+v\n", player.BaseInfo)
			player.bulletRates = make([]*data.BulletRate, 0, len(room.bulletRates))
			for _, b := range room.bulletRates {

				if b.Gun <= player.BaseInfo.MaxPower {
					player.bulletRates = append(player.bulletRates, b)
					//log.Debug("bulletRates %+v\n", b)

				}
			}
		}
	}
	//log.Debug("player bulletRates %+v\n", player.bulletRates)

	newLen := 0
	if player.bulletRates != nil {
		newLen = len(player.bulletRates)
	}
	return newLen > oldLen
}
