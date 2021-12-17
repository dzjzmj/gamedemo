package base

import (
	"lolGF/conn"
	"lolGF/log"
)

// ----处理发送消息给客户端----

func SendPongToClient(uid int32) {
	conn.SendPongToClient(uid)
}

func SendMsgToGateEx(receiver int32, cmd int16, extraData interface{}, printLog bool) []byte {
	conn.SendMsgToClient(receiver, cmd, extraData, false)

	if printLog {
		log.Debug("send u:%d c:%d %v", receiver, cmd, extraData)
	}
	return nil
}

func SendConstDataToGate(receiver int32, data []byte) {
	conn.SendDataToClient(receiver, data)
}

func SendMsgToGate(receiver int32, cmd int16, extraData interface{}) []byte {
	return SendMsgToGateEx(receiver, cmd, extraData, true)
}

type Ret struct {
	Ret int
}

func SendRetToGate(receiver int32, cmd int16, ret int) []byte {
	extraData := Ret{ret}
	return SendMsgToGateEx(receiver, cmd, extraData, true)
}
