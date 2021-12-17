package main

import (
	"fish_server_2021/common"
	"fish_server_2021/game/base"
	"fish_server_2021/game/data"
	"fish_server_2021/game/data/model"
	"fish_server_2021/game/gameLogic"
	"fish_server_2021/libs/database"
	"fish_server_2021/libs/redis"
	"fmt"
	"github.com/json-iterator/go"
	"lolGF/log"
	"lolGF/module"
	"os"

	"lolGF/conn"
	"lolGF/env"
)

func main() {
	env.BinInfo.Version = "1.0.0"
	env.BinInfo.VersionDesc = "游戏服务1.0"
	conn.LogClientMsg = true

	for _, v := range os.Args {
		if v == "-v" {
			vInfo, _ := jsoniter.MarshalToString(env.BinInfo)
			fmt.Println(vInfo)
			return
		}
	}
	log.InitLog(base.BaseConf.LogPath, base.BaseConf.LogLevel, base.BaseConf.LogMaxSize)
	if err := redis.Init(base.BaseConf.RedisAddr, base.BaseConf.RedisPwd, 0); err != nil {
		panic(err)
	}
	if err := database.InitDB(base.BaseConf.DBAcc, base.BaseConf.DBURL, 5, 5); err != nil {
		panic(err)
	}
	if err := database.InitAdminDB(base.AdminConf.DBAcc, base.AdminConf.DBURL, 5, 5); err != nil {
		panic(err)
	}
	data.AutoMigrate()
	data.InitRoomData()
	common.InitConfigData()
	model.RWorkers = model.NewRWorkers()
	module.Run(
		"GameServer",
		gameLogic.Module,
	)

	conn.DisconnectNats()
}
