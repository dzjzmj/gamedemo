package base

import (
	"lolGF/utils/eio"
)

var (
	StopServerTime int64
)

// BaseConf 基础配置,支付、后台等
var BaseConf struct {
	LogPath    string
	LogLevel   string
	LogMaxSize int
	DBAcc      string //账号服数据库账号密码 eg:"user:password"
	DBURL      string

	RedisAddr string
	RedisPwd  string
}
var AdminConf struct {
	DBAcc string //账号服数据库账号密码 eg:"user:password"
	DBURL string
}

func init() {
	err := eio.ReadJSONFile("conf/game-server.conf", &BaseConf)
	if err != nil {
		panic("read game-server.conf fail")
	}
	err = eio.ReadJSONFile("conf/admin-server.conf", &AdminConf)
	if err != nil {
		panic("read admin-server.conf fail")
	}
}
