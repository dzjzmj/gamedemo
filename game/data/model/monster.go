package model

import "fish_server_2021/libs/database"

type MonsterGrow struct {
	Id       int
	Room     int
	Type     int
	Monster  string
	Weight   string
	Num      int
	Interval int
	Appear   int
	Loop     int
	MaxNum   int32
	MinNum   int32
}

func GetMonsterGrowByRoom(roomType int) []MonsterGrow {
	var list []MonsterGrow
	if roomType > 0 {
		database.AdminDB.Order("id asc").Where("room=?", roomType).Find(&list)
	} else {
		database.AdminDB.Order("id asc").Find(&list)
	}

	return list
}
