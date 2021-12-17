package internal

import (
	"fish_server_2021/game/base"
	"fish_server_2021/game/data"
	"time"
)

const (
	MonsterStatusLive = 1
	MonsterStatusDeal = 2
)

type Monster struct {
	ID              int64
	FormationConfig *data.MonsterFormationConfig
	GrowTime        int64
	status          int // 状态
	config          *MonsterGrowConfig
	Live            int64
	buffTime        int64
	monsterData     *data.Monster
}

type MonsterRet []interface{} // 返回给客户端[怪物实例ID, 怪物阵形表ID,产生时间]
func (room *Room) sendMonsterRet(uid int32, monster *Monster, calTime bool) {
	var sec int64
	if calTime {
		currMillisecond := time.Now().UnixNano() / 1000 / 1000
		sec = currMillisecond - monster.GrowTime
	}
	ret := MonsterRet{monster.ID, monster.FormationConfig.Id, sec}
	if uid > 0 {
		base.SendMsgToGateEx(uid, SMGrowMaster, ret, true)
	} else {
		room.notifyAll(SMGrowMaster, ret)
	}
}

func (room *Room) sendMonstersRet(uid int32) {
	room.configMonsterNumLock.RLock()

	rets := make([]MonsterRet, 0, 100)
	currMillisecond := time.Now().UnixNano() / 1000 / 1000
	for _, monster := range room.monsters {
		sec := (currMillisecond - monster.GrowTime) + monster.buffTime*1000
		ret := MonsterRet{monster.ID, monster.FormationConfig.Id, sec}
		rets = append(rets, ret)

	}
	room.configMonsterNumLock.RUnlock()

	if uid > 0 {
		base.SendMsgToGateEx(uid, SMGrowMasters, rets, true)
	} else {
		room.notifyAll(SMGrowMasters, rets)
	}
}
