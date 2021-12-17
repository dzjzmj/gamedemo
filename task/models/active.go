package models

import (
	"fish_server_2021/libs/database"
	"time"
)

const ActiveTypeDay = 1
const ActiveTypeWeek = 2
const ActiveTypeAchieve = 3

type ActiveConfig struct {
	ID           int
	LivenessType int
	Stage        int
	Award        string
	AddLiveness  int64
}

const PlayerActiveStatusWait = 0
const PlayerActiveStatusFinish = 1
const PlayerActiveStatusSuccess = 2

type PlayerActive struct {
	ID           int64
	LivenessType int
	Stage        int
	Status       int // 0进行中 1完成 2已领取
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (m *PlayerActive) Save() {
	if m.ID == 0 {
		database.DB.Create(m)
	} else {
		database.DB.Save(m)
	}
}

func GetActiveConfigs() []ActiveConfig {
	var list []ActiveConfig
	database.DB.Order("stage asc").Find(&list)
	return list
}
func GetPlayerActives() []*PlayerActive {
	var list []*PlayerActive
	database.DB.Order("stage asc").Find(&list)
	return list
}
