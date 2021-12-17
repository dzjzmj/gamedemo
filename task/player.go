package main

import (
	"fish_server_2021/common"
	"fish_server_2021/task/models"
	"lolGF/utils/econv"
	"sort"
	"strings"
	"sync"
	"time"
)

var userMap sync.Map

type Player struct {
	BaseInfo          *common.UserInfo
	TaskLock          sync.RWMutex
	Tasks             map[int]*models.PlayerTask
	quitTimer         *time.Timer
	maxTaskId         int
	typeOnlineTimeNum int

	activeData [][]*models.PlayerActive
}

func getRunObjectiveType(uid int32) map[int]int {
	player := getUser(uid)
	player.TaskLock.RLock()
	defer player.TaskLock.RUnlock()
	var tasks map[int]int
	player.typeOnlineTimeNum = 0
	for _, pt := range player.Tasks {
		if pt.Status == models.PlayerTaskStatusUnFinish {
			task := GetTaskById(pt.TaskId)
			if task != nil {
				if tasks == nil {
					tasks = make(map[int]int)
				}
				tasks[task.ObjectiveType]++
				if task.ObjectiveType == common.TaskObjectiveTypeOnline {
					player.typeOnlineTimeNum++
				}
			}
		}
	}
	return tasks
}

// 计算在线时间 每分钟执行一次
func calOnlineTime() {
	userMap.Range(func(key, value interface{}) bool {
		player := value.(*Player)
		if player != nil && player.typeOnlineTimeNum > 0 {
			ret := triggerTask(player.BaseInfo.ID, common.TaskObjectiveTypeOnline, "1")
			if ret.UnFinishNum <= 0 {
				player.typeOnlineTimeNum = 0
			}
		}
		return true
	})
	return
}

func (player *Player) hasTask(id int) (*models.PlayerTask, bool) {
	player.TaskLock.RLock()
	defer player.TaskLock.RUnlock()
	playerTask, ok := player.Tasks[id]
	return playerTask, ok
}
func (player *Player) saveTasks() {
	player.TaskLock.Lock()
	defer player.TaskLock.Unlock()

	for _, task := range player.Tasks {
		task.Save()
	}
}

func (player *Player) deleteTask(id int) {
	player.TaskLock.Lock()
	defer player.TaskLock.Unlock()
	delete(player.Tasks, id)
}

type PlayerActiveInfo struct {
	List   []PlayerActive
	Active int64
}

func (player *Player) getActiveList() map[int]*PlayerActiveInfo {
	ret := make(map[int]*PlayerActiveInfo)
	for i, list := range player.activeData {
		if i > 0 {
			for _, ac := range list {
				tmp := PlayerActive{
					ID:           ac.ID,
					LivenessType: ac.LivenessType,
					Stage:        ac.Stage,
					Status:       ac.Status,
					Config:       GetActiveConfigByStage(ac.LivenessType, ac.Stage),
				}
				if ret[i] == nil {
					ret[i] = &PlayerActiveInfo{
						List:   make([]PlayerActive, 0, 10),
						Active: player.BaseInfo.AchieveActive,
					}
					if i == models.ActiveTypeDay {
						ret[i].Active = player.BaseInfo.DayActive
					}
					if i == models.ActiveTypeWeek {
						ret[i].Active = player.BaseInfo.WeekActive
					}
				}
				ret[i].List = append(ret[i].List, tmp)
			}
		}
	}
	return ret
}

func (player *Player) getList() PlayerTasks {
	player.TaskLock.RLock()
	tmp := make(PlayerTasks, 0, 100)
	for _, task := range player.Tasks {
		if task.Status == models.PlayerTaskStatusWait {
			continue
		}
		t := GetTaskById(task.TaskId)
		tmp = append(tmp, PlayerTask{
			ID:         task.ID,
			Uid:        task.Uid,
			Task:       t,
			FinishData: task.FinishData,
			Status:     task.Status,
		})
	}
	player.TaskLock.RUnlock()
	sort.Sort(tmp)
	return tmp
}

func (player *Player) getRoomTask(roomType int, nextId int) *PlayerTask {
	if nextId > 0 {
		pt, ok := player.hasTask(nextId)
		if ok {
			task := GetTaskById(pt.TaskId)
			return &PlayerTask{
				ID:         pt.ID,
				Uid:        pt.Uid,
				Task:       task,
				FinishData: pt.FinishData,
				Status:     pt.Status,
			}
		}
	}
	player.TaskLock.RLock()
	defer player.TaskLock.RUnlock()
	for _, pt := range player.Tasks {

		task := GetTaskById(pt.TaskId)

		if task.ShowRoom != roomType {
			continue
		}

		if pt.Status == models.PlayerTaskStatusFinish || pt.Status == models.PlayerTaskStatusUnFinish {
			return &PlayerTask{
				ID:         pt.ID,
				Uid:        pt.Uid,
				Task:       task,
				FinishData: pt.FinishData,
				Status:     pt.Status,
			}
		}
	}
	return nil
}

func (player *Player) triggerTask(playerTask *models.PlayerTask, objectiveData string) bool {

	task := GetTaskById(playerTask.TaskId)

	var needNum int64

	if task.ObjectiveType == common.TaskObjectiveTypeOnline {
		playerTask.FinishData += 1
		needNum, _ = econv.Int64(task.ObjectiveData)
	} else if task.ObjectiveType == common.TaskObjectiveTypeAd {
		playerTask.FinishData += 1
		needNum, _ = econv.Int64(task.ObjectiveData)
	} else if task.ObjectiveType == common.TaskObjectiveTypeLogin {
		n, _ := econv.Int64(objectiveData)
		playerTask.FinishData += n
		needNum, _ = econv.Int64(task.ObjectiveData)

	} else if task.ObjectiveType == common.TaskObjectiveTypeKillMonster {
		configs := strings.Split(task.ObjectiveData, ",")
		if len(configs) == 4 {
			needId, _ := econv.Int(configs[0])
			needRate, _ := econv.Int64(configs[1])
			needType, _ := econv.Int(configs[2])
			needNum, _ = econv.Int64(configs[3])

			datas := strings.Split(objectiveData, ",")
			if len(datas) == 4 {
				id, _ := econv.Int(datas[0])
				rate, _ := econv.Int64(datas[1])
				typ, _ := econv.Int(datas[2])
				n, _ := econv.Int64(datas[3])
				if needId > 0 && id != needId {
					return false
				}
				if needRate > 0 && rate < needRate {
					return false
				}
				if needType > 0 && typ != needType {
					return false
				}

				playerTask.FinishData += n
			}
		}

	} else if task.ObjectiveType == common.TaskObjectiveTypeShoot {
		//n, _ := econv.Int64(objectiveData)
		playerTask.FinishData += 1
		needNum, _ = econv.Int64(task.ObjectiveData)
	} else if task.ObjectiveType == common.TaskObjectiveTypeUseItem {
		configs := strings.Split(task.ObjectiveData, ",")
		if len(configs) == 3 {
			needItemId, _ := econv.Int32(configs[0])
			needItemType, _ := econv.Int32(configs[1])
			needNum, _ = econv.Int64(configs[2])
			datas := strings.Split(objectiveData, ",")
			if len(datas) == 3 {
				itemId, _ := econv.Int32(datas[0])
				itemType, _ := econv.Int32(datas[1])
				if needItemType > 0 && itemType != needItemType {
					return false
				}
				if needItemId > 0 && itemId != needItemId {
					return false
				}
				n, _ := econv.Int64(datas[2])
				playerTask.FinishData += n
			}
		}
	} else if task.ObjectiveType == common.TaskObjectiveTypeAddItem {

		configs := strings.Split(task.ObjectiveData, ",")
		//log.Debug("triggerTask1 %+v", configs)

		if len(configs) == 3 {
			needItemId, _ := econv.Int32(configs[0])
			needItemType, _ := econv.Int32(configs[1])
			needNum, _ = econv.Int64(configs[2])
			datas := strings.Split(objectiveData, ",")
			//log.Debug("triggerTask2 %+v", datas)

			if len(datas) == 3 {
				itemId, _ := econv.Int32(datas[0])
				itemType, _ := econv.Int32(datas[1])
				//log.Debug("triggerTask3 %v %v %v %v", itemId, needItemId, itemType, needItemType)
				if needItemType > 0 && itemType != needItemType {
					return false
				}
				if needItemId > 0 && itemId != needItemId {
					return false
				}
				n, _ := econv.Int64(datas[2])
				playerTask.FinishData += n
			}
		}
	} else if task.ObjectiveType == common.TaskObjectiveTypeUpPower {
		n, _ := econv.Int64(objectiveData)
		playerTask.FinishData = n
		needNum, _ = econv.Int64(task.ObjectiveData)
	} else if task.ObjectiveType == common.TaskObjectiveTypeRedPack {
		configs := strings.Split(task.ObjectiveData, ",")
		if len(configs) == 2 {
			needItemId, _ := econv.Int(configs[0])
			needNum, _ = econv.Int64(configs[1])

			id, _ := econv.Int(objectiveData)

			if needItemId > 0 && needItemId != id {
				return false
			}
			playerTask.FinishData += 1
		}
	}
	//log.Debug("triggerTask4 %+v %v", playerTask, needNum)
	if playerTask.FinishData >= needNum {
		playerTask.Status = models.PlayerTaskStatusFinish
		playerTask.Save()

		return true
	}
	return false
}

func getCacheUser(uid int32) *Player {
	value, ok := userMap.Load(uid)
	if !ok || value == nil {
		return nil
	}
	one := value.(*Player)
	return one
}
func delCacheUser(uid int32) {
	userMap.Delete(uid)
}

func getUser(uid int32) *Player {
	player := getCacheUser(uid)
	if player == nil {
		player = &Player{
			BaseInfo:   nil,
			Tasks:      make(map[int]*models.PlayerTask),
			activeData: make([][]*models.PlayerActive, 4),
		}
		player.activeData[1] = make([]*models.PlayerActive, 0, 10)
		player.activeData[2] = make([]*models.PlayerActive, 0, 10)
		player.activeData[3] = make([]*models.PlayerActive, 0, 10)
		player.getDbActiveData()

		tasks := models.GetPlayerTask(uid)
		for _, task := range tasks {
			if task.TaskId > player.maxTaskId {
				player.maxTaskId = task.TaskId
			}
			player.Tasks[task.TaskId] = task
		}
		userMap.Store(uid, player)
	}
	player.BaseInfo = common.GetUserInfo(uid)

	return player
}
