package model

import "fish_server_2021/libs/database"

type LevelConfig struct {
	ID    int
	Exp   int64
	Award string
}

func AllLevelConfigs() []LevelConfig {
	var list []LevelConfig
	database.AdminDB.Order("id asc").Find(&list)
	return list
}
