package data

import (
	"fish_server_2021/game/data/model"
	"fish_server_2021/libs/database"
	"time"
)

func AutoMigrate() {
	_ = database.DB.AutoMigrate(new(model.PlayerDayCount), new(model.GameRecord))
	_ = database.AdminDB.AutoMigrate(new(model.LevelConfig), new(model.Pool), new(PoolItem),
		new(Monster), new(model.RoomType), new(model.MonsterGrow), new(BulletRate),
	)
}

func CreateGameRecord(record *model.GameRecord) {
	record.CreatedAt = time.Now()
	model.RWorkers.Push(record)
	return
}

func GetAllMonster() []Monster {
	var list []Monster
	database.AdminDB.Find(&list)
	return list
}

func GetAllBulletRate() []BulletRate {
	var list []BulletRate
	database.AdminDB.Find(&list)
	return list
}
