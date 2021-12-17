package model

import "fish_server_2021/libs/database"

type RoomType struct {
	Id             int
	Name           string
	RoomType       int
	IncludePower   string
	Describe       string
	MapID          string
	NeedVip        int
	NeedMoney      int64
	NeedPower      int64
	MonsterRevenue float64
	RedpackRevenue float64
	SpecialRevenue float64
	BossRevenue    float64
	ChestRevenue   float64
}

func GetAllRoomType() []RoomType {
	var list []RoomType
	database.AdminDB.Find(&list)
	return list
}
