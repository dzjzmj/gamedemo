package model

import (
	"fish_server_2021/libs/database"
	"time"
)

type PlayerDayCount struct {
	ID              int64
	UID             int32  `gorm:"index:idx_date_uid"`
	Date            string `gorm:"size:10;index:idx_date_uid"`
	CoinAddSum      int64  //累计金币奖励(扣税)
	CoinAddOriginal int64  //累计金币奖励
	CoinDecSum      int64  //累计金币消耗
	Pool            int64
	DropItem        string
	UpdateTime      int64
	Day             int
	CreateAt        time.Time
}

func (item *PlayerDayCount) Create() error {

	return database.DB.Create(item).Error
}

func (item *PlayerDayCount) Save() error {

	return database.DB.Save(item).Error
}

func GetPlayerDayCount(date string, uid int32) PlayerDayCount {
	var one PlayerDayCount
	one.Date = date
	one.UID = uid
	database.DB.Where("date = ? and uid=?", date, uid).First(&one)

	return one
}
