package model

import "fish_server_2021/libs/database"

type Pool struct {
	ID       int32
	PoolId   int
	RoomType int
	Pool     int64
}

func GetPool(poolId int, roomType int) Pool {
	var one Pool
	one.PoolId = poolId
	one.RoomType = roomType
	database.AdminDB.Where("pool_id=? and room_type=?", poolId, roomType).First(&one)

	if one.ID == 0 {
		database.AdminDB.Create(&one)
	}
	return one
}

func SavePool(poolId int, roomType int, pool int64) {
	var p Pool
	database.AdminDB.Model(&p).
		Where("pool_id=? and room_type=?", poolId, roomType).
		Update("pool", pool)
}
