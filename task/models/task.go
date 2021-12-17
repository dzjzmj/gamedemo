package models

import (
	"fish_server_2021/libs/database"
	"time"
)

const TaskTypeDay = 1

type Task struct {
	ID            int
	NextId        int
	Name          string
	Desc          string
	Icon          string
	Order         int
	TaskType      int
	ObjectiveType int
	ObjectiveData string
	ValidRoom     string
	Award         string
	DoubleAward   int
	Skip          int
	SkipData      string
	Advertising   int
	Status        int
	ShowRoom      int
}

const PlayerTaskStatusWait = 0
const PlayerTaskStatusUnFinish = 1
const PlayerTaskStatusFinish = 2
const PlayerTaskStatusSuccess = 3
const PlayerTaskStatusCancel = 4

type PlayerTask struct {
	ID         int64
	Uid        int32
	TaskId     int
	FinishData int64
	IsShow     int // 1显示 0不显示
	Status     int // 0待领取 1待完成 2完成 3已领取
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (m *PlayerTask) Save() {
	if m.ID == 0 {
		database.DB.Create(m)
	} else {
		database.DB.Save(m)
	}
}

func GetAllTask() []*Task {
	var list []*Task
	database.DB.Order("`order` desc").Find(&list)
	return list
}

func GetPlayerTask(uid int32) []*PlayerTask {
	var list []*PlayerTask
	database.DB.Where("uid=? and is_show=1", uid).Find(&list)
	return list
}
