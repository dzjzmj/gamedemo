package internal

import (
	"errors"
	"fish_server_2021/common"
	"fish_server_2021/game/data"
	"fmt"
)

type GameRPC int

func (*GameRPC) UpdatePool(roomId int, pools map[int]*data.Pools, ret *bool) error {
	data.SetAllPools(roomId, pools)
	//fmt.Printf("%v \n %+v",roomId, pools[104])
	*ret = true
	return nil
}
func (*GameRPC) GetPool(roomId int, pools *map[int]*data.Pools) error {
	tmp, ok := data.GetAllPools(roomId)
	if ok {
		*pools = tmp
	} else {
		return errors.New("pool not exist")
	}
	return nil
}

func (*GameRPC) UpdateData(tableName string, roomType int, ret *bool) error {
	fmt.Println(tableName, roomType)
	if tableName == "monsters" {
		data.GetMonsterToMap()
	} else if tableName == "bullet_rates" {
		data.GetBulletRateToMap()
	} else if tableName == "level_configs" {
		data.InitLevelData()
	} else if tableName == "room_types" {
		data.GetRoomTypeToMap()
	} else if tableName == "redpacks" {
		data.InitRadPack()
	} else if tableName == "init_configs" {
		common.InitConfigData()
		common.AccountUpdateData(tableName, roomType)
	} else if tableName == "monster_grows" {
		isChange := data.GetMonsterGrowByRoom(roomType)
		if isChange {
			MatchRoom.Range(func(key, value interface{}) bool {
				room := value.(*Room)
				if room.RoomType.Id == roomType {
					room.initGrowMonster()
				}
				return true
			})
		}
	}
	*ret = true
	return nil
}
