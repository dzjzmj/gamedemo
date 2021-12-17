package main

import (
	"fish_server_2021/libs/database"
	"fish_server_2021/task/models"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/robfig/cron"
	"lolGF/conn"
	"lolGF/env"
	"lolGF/log"
	"lolGF/rpc"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	env.BinInfo.Version = "1.0.0"
	env.BinInfo.VersionDesc = "任务服务1.0"
	conn.LogClientMsg = true

	for _, v := range os.Args {
		if v == "-v" {
			vInfo, _ := jsoniter.MarshalToString(env.BinInfo)
			fmt.Println(vInfo)
			return
		}
	}
	log.InitLog(ServConf.LogPath, ServConf.LogLevel, ServConf.LogMaxSize)

	if err := database.InitDB(ServConf.DBAcc, ServConf.DBURL, 5, 5); err != nil {
		panic(err)
	}

	_ = database.DB.AutoMigrate(new(models.Task), new(models.PlayerTask), new(models.PlayerActive), new(models.ActiveConfig))
	initData()

	conn.ServiceSubscribe(&conn.SubscribeInfo{
		Name:          "task",
		EventMap:      ClientHandler.EventMap(),
		ClientHandler: ClientHandler.ClientHandler,
	})
	conn.SubscribeEvent(conn.EventClientClose, func(data []byte) {
		var ipc conn.IPCForMS
		_, err := ipc.Unmarshal(data)
		if err != nil {
			return
		}
		uid := ipc.Sender
		player := getUser(uid)
		player.quitTimer = time.AfterFunc(time.Second*30, func() {
			player.saveTasks()
			delCacheUser(uid)
		})
	})

	err := rpc.ServerOverNats("", new(TaskRPC), nil)
	log.Debug("ServerOverNats %v", err)

	spec := "0 * * * * ?"
	c := cron.New()
	err1 := c.AddFunc(spec, func() {
		calOnlineTime()
	})
	if err1 != nil {
		panic(err1)
	}
	c.Start()

	<-stopChan // wait for SIGINT

	userMap.Range(func(key, value interface{}) bool {
		one := value.(*Player)
		one.saveTasks()
		return true
	})
	conn.UnSubscribeAll()
	log.Debug("\n")
	log.Debug("Shutting down task server...")
}
