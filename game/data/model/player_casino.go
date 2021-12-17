package model

import (
	"context"
	"fish_server_2021/libs/redis"
	"lolGF/utils/econv"
)

type PlayerCasino struct {
	ID           int32
	CurrentLevel int
	GotIds       uint32
}

func GetPlayerCasino(id int32) *PlayerCasino {
	uidStr, _ := econv.String(id)
	key := "PlayerCasino" + uidStr
	retMap := redis.RClient.HGetAll(context.Background(), key)
	var dbData *PlayerCasino
	rm, err := retMap.Result()
	if err == redis.Nil || (err == nil && len(rm) == 0) {
		for k, v := range rm {
			if k == "c" {
				dbData.CurrentLevel, _ = econv.Int(v)
			}
			if k == "g" {
				dbData.GotIds, _ = econv.Uint32(v)
			}
		}
	}
	return dbData
}
