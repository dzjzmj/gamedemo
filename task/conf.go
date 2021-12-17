package main

import "lolGF/utils/eio"

var ServConf servConf

type servConf struct {
	DBAcc string //账号服数据库账号密码 eg:"user:password"
	DBURL string

	RedisAddr string
	RedisPwd  string

	LogLevel   string //日志输出级别 eg:debug/info/warn/error
	LogPath    string //日志输出路径 eg:"log/account"
	LogMaxSize int
}

func init() {

	err := eio.ReadJSONFile("conf/task-server.conf", &ServConf)
	if err != nil {
		panic(err)
	}

}

type JoinRoomReq struct {
	RT int
}
type GetAwardReq struct {
	ID     int64
	TaskID int
}
type GetAwardRet struct {
	Ret   int
	Items []Item
}

type ActiveFinishRet struct {
	LivenessType int
	Stage        int
	Status       int // 0进行中 1完成 2已领取
}

type ActiveGetAwardReq struct {
	LivenessType int
	Stage        int
}
