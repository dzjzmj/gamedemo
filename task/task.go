package main

import (
	"fish_server_2021/common"
	"fish_server_2021/common/proto"
	"fish_server_2021/task/models"
	"lolGF/conn"
	"lolGF/log"
	"lolGF/utils/econv"
	"strings"
	"sync"
	"time"
)

var allTasks []*Task
var allTasksMap map[int]*Task
var allTaskMapLock sync.RWMutex
var taskNextToPer map[int]int // next => perId
var MaxTaskId int

func GetTaskById(id int) *Task {
	allTaskMapLock.RLock()
	defer allTaskMapLock.RUnlock()
	return allTasksMap[id]
}

type PlayerTask struct {
	ID         int64
	Uid        int32
	Task       *Task
	FinishData int64
	Status     int // 1待完成 2完成 3已领取
}

type PlayerTasks []PlayerTask

func (s PlayerTasks) Len() int {
	return len(s)
}
func (s PlayerTasks) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s PlayerTasks) Less(i, j int) bool {
	return s[i].Task.Order < s[j].Task.Order
}

type Task struct {
	ID            int
	NextId        int
	PerId         int `msgpack:"-"`
	Name          string
	Desc          string
	Icon          string
	Order         int `msgpack:"-"`
	TaskType      int
	ObjectiveType int
	ObjectiveData string
	ValidRoom     []int
	Award         []Item
	DoubleAward   int
	Skip          int
	SkipData      string
	Advertising   int
	ShowRoom      int
}
type Item struct {
	Id  int32 // 物品ID
	Num int64 // 数量
}

func initData() {
	allTasks = make([]*Task, 0, 100)
	taskNextToPer = make(map[int]int)
	allTasksMap = make(map[int]*Task)
	list := models.GetAllTask()
	for _, task := range list {
		if task.ID > MaxTaskId {
			MaxTaskId = task.ID
		}
		if task.NextId > 0 {
			taskNextToPer[task.NextId] = task.ID
		}
		tmp := Task{
			ID:            task.ID,
			NextId:        task.NextId,
			Name:          task.Name,
			Desc:          task.Desc,
			Icon:          task.Icon,
			Order:         task.Order,
			TaskType:      task.TaskType,
			ObjectiveType: task.ObjectiveType,
			ObjectiveData: task.ObjectiveData,
			DoubleAward:   task.DoubleAward,
			Skip:          task.Skip,
			SkipData:      task.SkipData,
			Advertising:   task.Advertising,
			ShowRoom:      task.ShowRoom,
		}
		tmp.Award = make([]Item, 0, 10)
		awards := strings.Split(task.Award, ";")
		for _, c := range awards {
			s := strings.Split(c, ",")
			if len(s) == 2 {
				id, _ := econv.Int32(s[0])
				num, _ := econv.Int64(s[1])
				item := Item{
					Id:  id,
					Num: num,
				}
				tmp.Award = append(tmp.Award, item)
			}
		}
		rs := strings.Split(task.ValidRoom, ",")
		tmp.ValidRoom = make([]int, len(rs))
		for k, i := range rs {
			room, _ := econv.Int(i)
			tmp.ValidRoom[k] = room
		}
		allTasks = append(allTasks, &tmp)

	}

	for _, task := range allTasks {
		task.PerId = taskNextToPer[task.ID]
		allTasksMap[task.ID] = task
	}
	initActiveConfigData()
}
func resetDayTask(player *Player) {
	player.TaskLock.RLock()

	nowStr := time.Now().Format("2006-01-02")
	addDayFlag := false
	delIds := make([]int, 0, 20)
	for _, pt := range player.Tasks {
		task := GetTaskById(pt.TaskId)
		if task.TaskType == models.TaskTypeDay {
			if pt.CreatedAt.Format("2006-01-02") != nowStr {
				pt.IsShow = 0
				pt.Save()
				delIds = append(delIds, task.ID)
				addDayFlag = true
			}
		}

	}
	player.TaskLock.RUnlock()

	for _, id := range delIds {
		player.deleteTask(id)
	}
	if addDayFlag {
		addTaskToPlayer(player)
	}
}
func addTaskToPlayer(player *Player) {
	for _, task := range allTasks {
		_, has := player.hasTask(task.ID)
		if !has {
			log.Debug("no has %v", task.ID)
			status := models.PlayerTaskStatusUnFinish
			if task.PerId > 0 {
				per, ok := player.hasTask(task.PerId)
				if ok && (per.Status == models.PlayerTaskStatusUnFinish || per.Status == models.PlayerTaskStatusWait) {
					status = models.PlayerTaskStatusWait
				} else {
					status = models.PlayerTaskStatusWait
				}
			}
			newTask := models.PlayerTask{
				ID:         0,
				Uid:        player.BaseInfo.ID,
				TaskId:     task.ID,
				FinishData: 0,
				Status:     status,
				IsShow:     1,
			}
			newTask.Save()
			player.TaskLock.Lock()
			player.Tasks[task.ID] = &newTask
			player.TaskLock.Unlock()
		} else {
			log.Debug("hasTask %v %v", player.BaseInfo.ID, task.ID)
		}
	}
}

func getAward(player *Player, req GetAwardReq) (*GetAwardRet, error) {
	playerTask, ok := player.hasTask(req.TaskID)
	if ok {
		if playerTask.ID == req.ID && playerTask.Status == models.PlayerTaskStatusFinish {
			playerTask.Status = models.PlayerTaskStatusSuccess
			playerTask.Save()

			task := GetTaskById(playerTask.TaskId)

			// 奖励
			itemProto := proto.ItemsReq{
				Uid:    player.BaseInfo.ID,
				Items:  make(map[int32]int64),
				Action: 6,
			}
			for _, item := range task.Award {
				itemProto.Items[item.Id] = item.Num
			}
			err := common.ItemClient.AddItems(itemProto)
			if err != nil {
				log.Debug("AddItems %v", err)
			} else {
				for _, item := range task.Award {
					if item.Id == common.DayActiveItemId || item.Id == common.WeekActiveItemId {
						addActiveNum(player, item.Id)
					}
				}
			}

			ret := &GetAwardRet{Items: task.Award, Ret: 1}

			// 串联任务 下一个处理
			var nextId int
			if task.NextId > 0 {
				nextTask, ok := player.hasTask(task.NextId)
				if ok {
					nTask := GetTaskById(nextTask.TaskId)

					nextTask.Status = models.PlayerTaskStatusUnFinish
					if nTask.ObjectiveType == common.TaskObjectiveTypeUpPower {
						p, _ := econv.String(player.BaseInfo.MaxPower)
						if player.triggerTask(nextTask, p) {
							nextTask.Status = models.PlayerTaskStatusFinish
						}
					}

					nextTask.Save()
					nextId = nTask.ID
				}
			}

			// 下一个任务
			if task.ShowRoom > 0 {
				pt := player.getRoomTask(task.ShowRoom, nextId)
				if pt != nil {
					conn.SendMsgToClient(player.BaseInfo.ID, common.CMNewTask, pt, true)
				}
			}
			return ret, nil
		}
	}
	return nil, nil
}
