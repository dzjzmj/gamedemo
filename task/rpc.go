package main

import (
	"fish_server_2021/common"
	"fish_server_2021/task/models"
	"lolGF/conn"
)

type TaskRPC int

func (*TaskRPC) GetRunObjectiveType(uid int32, data *map[int]int) error {
	*data = getRunObjectiveType(uid)
	return nil
}

func (*TaskRPC) Trigger(uid int32, objectiveType int, objectiveData string, ret *common.TaskTriggerRet) error {
	*ret = triggerTask(uid, objectiveType, objectiveData)
	return nil
}

func (*TaskRPC) UpdateData(tableName string, ret *bool) error {
	if tableName == "tasks" {
		initData()
	}
	*ret = true
	return nil
}

func triggerTask(uid int32, objectiveType int, objectiveData string) common.TaskTriggerRet {
	player := getUser(uid)
	player.TaskLock.RLock()
	defer player.TaskLock.RUnlock()
	n := 0
	ret := false
	for _, pt := range player.Tasks {
		task := GetTaskById(pt.TaskId)

		if pt.Status == models.PlayerTaskStatusUnFinish && task.ObjectiveType == objectiveType {
			ret = player.triggerTask(pt, objectiveData)
			if ret {
				conn.SendMsgToClient(player.BaseInfo.ID, common.CMNewTask, PlayerTask{
					ID:         pt.ID,
					Uid:        pt.Uid,
					Task:       task,
					FinishData: pt.FinishData,
					Status:     pt.Status,
				}, true)
			}
			n++
		}
	}
	return common.TaskTriggerRet{UnFinishNum: n, IsFinish: ret}
}
