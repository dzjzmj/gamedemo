package model

import "fish_server_2021/libs/database"

type Redpack struct {
	Id          int
	Name        string
	Class       int
	AddNumber   int64
	AwardItem   string
	Probability string
	Tips        string
}

func GetRedPacks() []Redpack {
	var list []Redpack
	database.AdminDB.Order("id asc").Find(&list)
	return list
}
