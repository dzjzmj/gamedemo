package main

import (
	"fish_server_2021/common"
	"fish_server_2021/common/proto"
	"fish_server_2021/task/models"
	"lolGF/conn"
	"lolGF/log"
	"lolGF/utils/econv"
	"strings"
	"sync"
)

var activeConfigMap map[int][]*ActiveConfig
var activeConfigMapLock sync.RWMutex

func GetActiveConfig(typ int) []*ActiveConfig {
	activeConfigMapLock.RLock()
	ret := activeConfigMap[typ]
	activeConfigMapLock.RUnlock()
	return ret
}

func GetActiveConfigByStage(typ int, stage int) *ActiveConfig {
	list := GetActiveConfig(typ)
	for _, c := range list {
		if c.Stage == stage {
			return c
		}
	}
	return nil
}

type PlayerActive struct {
	ID           int64
	LivenessType int
	Stage        int
	Status       int // 0进行中 1完成 2已领取
	Config       *ActiveConfig
}
type ActiveConfig struct {
	ID           int
	LivenessType int
	Stage        int
	Award        []Item
	AddLiveness  int64
}

func initActiveConfigData() {
	activeConfigMap = make(map[int][]*ActiveConfig)

	list := models.GetActiveConfigs()
	for _, active := range list {
		tmp := &ActiveConfig{
			ID:           active.ID,
			LivenessType: active.LivenessType,
			Stage:        active.Stage,
			AddLiveness:  active.AddLiveness,
		}
		tmp.Award = make([]Item, 0, 10)
		awards := strings.Split(active.Award, ";")
		for _, c := range awards {
			s := strings.Split(c, ",")
			if len(s) == 2 {
				id, _ := econv.Int32(s[0])
				num, _ := econv.Int64(s[1])
				item := Item{
					Id:  id,
					Num: num,
				}
				tmp.Award = append(tmp.Award, item)
			}
		}

		_, ok := activeConfigMap[tmp.LivenessType]
		if !ok {
			activeConfigMap[tmp.LivenessType] = make([]*ActiveConfig, 0, 10)
		}
		activeConfigMap[tmp.LivenessType] = append(activeConfigMap[tmp.LivenessType], tmp)
	}
}

func addActiveNum(player *Player, itemId int32) {
	itemIdToType := map[int32]int{
		common.DayActiveItemId:     1,
		common.WeekActiveItemId:    2,
		common.AchieveActiveItemId: 3,
	}
	typ := itemIdToType[itemId]
	// 日活跃处理
	newNum, _ := common.ItemClient.GetItem(proto.ItemReq{
		Uid:    player.BaseInfo.ID,
		ItemId: itemId,
	})
	if newNum > 0 {
		if typ == models.ActiveTypeDay {
			player.BaseInfo.DayActive = newNum
		}
		if typ == models.ActiveTypeWeek {
			player.BaseInfo.WeekActive = newNum
		}
		if typ == models.ActiveTypeAchieve {
			player.BaseInfo.AchieveActive = newNum
		}
		if len(player.activeData[typ]) == 0 {
			player.initPlayerActiveData(typ)
		}

		for _, active := range player.activeData[typ] {
			if active.Status == models.PlayerActiveStatusWait {
				// 判断是否完成
				c := GetActiveConfigByStage(typ, active.Stage)
				if newNum >= c.AddLiveness {
					active.Status = models.PlayerActiveStatusFinish
					active.Save()
					// 发消息
					ret := ActiveFinishRet{
						LivenessType: active.LivenessType,
						Stage:        active.Stage,
						Status:       active.Status,
					}
					conn.SendMsgToClient(player.BaseInfo.ID, common.SMActiveChange, ret, false)
					break
				}
			}
		}
	}

}

func getActiveAward(player *Player, req ActiveGetAwardReq) (*GetAwardRet, error) {
	list := player.activeData[req.LivenessType]
	for _, act := range list {
		if act.Stage == req.Stage {
			if act.Status == models.PlayerActiveStatusFinish {
				act.Status = models.PlayerActiveStatusSuccess
				act.Save()

				// 奖励
				itemProto := proto.ItemsReq{
					Uid:   player.BaseInfo.ID,
					Items: make(map[int32]int64),
				}
				config := GetActiveConfigByStage(act.LivenessType, act.Stage)
				for _, item := range config.Award {
					itemProto.Items[item.Id] = item.Num
				}
				err := common.ItemClient.AddItems(itemProto)
				if err != nil {
					log.Debug("AddItems %v", err)
				}
				ret := &GetAwardRet{Items: config.Award, Ret: 1}
				return ret, nil
			}
		}
	}
	return nil, nil
}

func (player *Player) initPlayerActiveData(typ int) {
	configs := GetActiveConfig(typ)
	for _, config := range configs {
		tmp := &models.PlayerActive{
			LivenessType: config.LivenessType,
			Stage:        config.Stage,
		}
		tmp.Save()
		player.activeData[typ] = append(player.activeData[typ], tmp)
	}
}

func (player *Player) getDbActiveData() {
	list := models.GetPlayerActives()
	for _, pa := range list {
		player.activeData[pa.LivenessType] = append(player.activeData[pa.LivenessType], pa)
	}
	keys := map[int]bool{
		1: true,
		2: true,
		3: true,
	}
	for k := range keys {
		if len(player.activeData[k]) == 0 {
			player.initPlayerActiveData(k)
		}
	}
}
