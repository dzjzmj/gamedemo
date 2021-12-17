package internal

import (
	"fish_server_2021/common"
	"fish_server_2021/game/base"
	"fish_server_2021/game/data"
	"sync/atomic"
)

func (player *Player) newPlayerPoolCoefficient(monster *data.Monster, rate *data.BulletRate) (monsterWeight int32, playerWeight int32) {
	//log.Debug("LoginDay %d %v", player.BaseInfo.ID, player.BaseInfo.LoginDay)
	monsterWeight = -1
	playerWeight = 0
	if player.playerCount.Pool > 0 {
		monsterWeight = monster.RookieWeight

	}
	if player.BaseInfo.LoginDay > data.NewPlayerMaxDay {
		return
	}

	pool := player.playerCount.CoinAddSum
	pools, ok := data.GetNewPlayerPoolByDay(player.BaseInfo.LoginDay)
	if !ok {
		return
	}
	for _, p := range pools {
		if pool >= p.LowerLimit && pool <= p.UpperLimit {
			playerWeight = p.Coefficient
			return
		}
	}

	return
}
func (room *Room) changePool(monster *data.Monster, rate *data.BulletRate, num int64, player *Player) {
	pool := room.getPool(monster, rate)
	if pool == nil {
		return
	}
	// 公共水池
	atomic.AddInt64(&pool.Pool, num)
	if num < 0 {
		// 个人水池
		n := atomic.AddInt64(&player.playerCount.Pool, num)
		if n < 0 {
			player.playerCount.Pool = 0
		}
	}
}
func (room *Room) getPools(player *Player) {
	pp, _ := data.GetAllPools(room.RoomType.Id)
	base.SendMsgToGate(player.BaseInfo.ID, common.CMGetPool, pp)
	rets := make(map[int]int32)
	for _, item := range pp {
		rets[item.ID] = 0
		for _, p := range item.PoolItems {
			if item.Pool >= p.LowerLimit && item.Pool <= p.UpperLimit {
				rets[item.ID] = p.Coefficient
				break
			}
		}
	}
	base.SendMsgToGate(player.BaseInfo.ID, common.CMGetPool, rets)
}
func (room *Room) getPool(monster *data.Monster, rate *data.BulletRate) *data.Pools {

	var id int
	if monster.Type == data.MonsterTypeSmall {
		id = rate.MonsterPool
	} else if monster.Type == data.MonsterTypeRedPack {
		id = rate.RedpackPool
	} else if monster.Type == data.MonsterTypeLightning ||
		monster.Type == data.MonsterTypeBlackHole || monster.Type == data.MonsterTypeBomb {
		id = rate.SpecialPool
	} else if monster.Type == data.MonsterTypeBoss {
		id = rate.BossPool
	} else if monster.Type == data.MonsterTypePool {
		id = rate.ChestPool
	}
	if id > 0 {
		pool, ok := data.GetPools(room.RoomType.Id, id)
		if !ok {
			return nil
		}
		return pool
	}
	return nil
}
func (room *Room) getPoolCoefficient(monster *data.Monster, rate *data.BulletRate) int32 {
	pool := room.getPool(monster, rate)
	if pool != nil {
		if pool.Pool < rate.Gun*monster.Rate {
			return -totalWeight
		}
		for _, p := range pool.PoolItems {
			if pool.Pool >= p.LowerLimit && pool.Pool <= p.UpperLimit {
				return p.Coefficient
			}
		}

	}
	return 0
}
