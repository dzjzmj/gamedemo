package model

import (
	"fish_server_2021/libs/database"
	"time"
)

type GameRecord struct {
	ID        int64
	Uid       int32
	ItemId    int32
	Num       int64 // 数量 负数为减少
	CreatedAt time.Time
	Mid       int64
	MType     int
	Power     int64
}

var RWorkers *Workers

func NewRWorkers() *Workers {

	return NewWorkers(100, 100, func(data []interface{}) {

		if len(data) > 0 {

			var itemRecords = make([]*GameRecord, 0, len(data))

			for _, v := range data {
				itemRecords = append(itemRecords, v.(*GameRecord))
			}

			database.DB.CreateInBatches(itemRecords, len(itemRecords))
		}
	})
}
