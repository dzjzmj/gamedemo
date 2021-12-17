package main

import (
	"fish_server_2021/common"
	"fish_server_2021/task/models"
	"lolGF/conn"
	"lolGF/log"
	"lolGF/utils/econv"
	"time"
)

var ClientHandler *clientHandler

type ClientHandlerMap map[int16]func(uid int32, data interface{}) (interface{}, error)

type clientHandler struct {
	handlers ClientHandlerMap
}

func init() {

	ClientHandler = new(clientHandler)
	ClientHandler.handlers = make(ClientHandlerMap)
	ClientHandler.handlers[common.CMJoinRoom] = handleJoinRoom
	ClientHandler.handlers[common.CMGetTaskAward] = handleGetAward
	ClientHandler.handlers[common.CMActiveGetAward] = handleActiveGetAward
	ClientHandler.handlers[common.CMTaskList] = handleList
	ClientHandler.handlers[conn.CMConnect] = handleConnect

}

type listRet struct {
	Ret  int
	List PlayerTasks
}

func handleList(uid int32, data interface{}) (interface{}, error) {
	player := getUser(uid)

	tmp := player.getList()

	ret := make(PlayerTasks, 0, 100)
	for _, task := range tmp {
		if task.Status == models.PlayerTaskStatusFinish {
			ret = append(ret, task)
		}
	}
	for _, task := range tmp {
		if task.Status != models.PlayerTaskStatusFinish {
			ret = append(ret, task)
		}
	}

	activeList := player.getActiveList()
	conn.SendMsgToClient(uid, common.SMActiveList, activeList, true)
	return listRet{List: ret, Ret: 1}, nil
}

// 接收任务
func handleConnect(uid int32, data interface{}) (interface{}, error) {
	player := getUser(uid)
	if player.quitTimer != nil {
		player.quitTimer.Stop()
	}
	if MaxTaskId > player.maxTaskId {
		log.Debug("hc MaxTaskId %v %v", MaxTaskId, player.maxTaskId)
		addTaskToPlayer(player)
	}
	// 重置每日任务
	resetDayTask(player)
	// 完成登录任务
	triggerTask(uid, common.TaskObjectiveTypeLogin, "1")
	getRunObjectiveType(uid)
	return nil, nil
}

func handleGetAward(uid int32, data interface{}) (interface{}, error) {
	var req GetAwardReq
	if !conn.DecodeClientData(data, &req) {
		return nil, nil
	}
	player := getUser(uid)

	return getAward(player, req)
}
func handleActiveGetAward(uid int32, data interface{}) (interface{}, error) {
	var req ActiveGetAwardReq
	if !conn.DecodeClientData(data, &req) {
		return nil, nil
	}
	player := getUser(uid)

	return getActiveAward(player, req)
}
func handleJoinRoom(uid int32, data interface{}) (interface{}, error) {
	var req JoinRoomReq
	if !conn.DecodeClientData(data, &req) {
		return nil, nil
	}
	player := getUser(uid)
	ret := player.getRoomTask(req.RT, 0)

	if ret != nil {
		if ret.Task.ObjectiveType == common.TaskObjectiveTypeUpPower {
			playerTask, _ := player.hasTask(ret.Task.ID)
			p, _ := econv.String(player.BaseInfo.MaxPower)
			if player.triggerTask(playerTask, p) {
				ret.Status = models.PlayerTaskStatusFinish
			}
		}
		time.AfterFunc(time.Second, func() {
			conn.SendMsgToClient(uid, common.CMNewTask, ret, true)
		})
	}
	return nil, nil
}

func (c *clientHandler) EventMap() map[int16]bool {

	var res = make(map[int16]bool)

	for k := range c.handlers {
		res[k] = true
	}

	return res
}

func (c *clientHandler) ClientHandler(req conn.ClientRequestInfo, v interface{}) {

	ret, err := c.handlers[req.CMD](req.UID, v)
	if err != nil {

		result := common.Result{Ret: 1}
		result.Ret = 2 // 2失败 后续和客户端对协议
		result.Message = err.Error()
		conn.SendMsgToClient(req.UID, req.CMD, result, true)
	} else {
		if ret == nil {
			return
		}
		conn.SendMsgToClient(req.UID, req.CMD, ret, true)
	}

}
