package internal

import (
	"fish_server_2021/common"
	"fish_server_2021/common/proto"
	"fish_server_2021/game/data"
	"fish_server_2021/game/data/model"
	"lolGF/conn"
	"lolGF/log"
	"lolGF/module"
	"lolGF/rpc"
	"time"
)

var (
	skeleton      = module.NewSkeleton()
	ChanRPC       = skeleton.ChanRPCServer
	moduleIsClose = false
	currTimeInDay = time.Now().Day()
	currUnixTime  = time.Now().Unix()
)

type Module struct {
	*module.Skeleton
}

func (m *Module) OnInit() {
	m.Skeleton = skeleton

	ClientMsg := make(map[int16]bool)
	for cmd := range ClientCMD {
		ClientMsg[cmd] = true
	}
	//连接消息服务
	conn.ServiceSubscribe(&conn.SubscribeInfo{
		Name:          "GameBase",
		EventMap:      ClientMsg,
		ClientHandler: HandleClientMSG,
	})
	conn.SubscribeEvent(conn.EventClientClose, func(data []byte) {
		var ipc conn.IPCForMS
		_, err := ipc.Unmarshal(data)
		if err != nil {
			return
		}
		//log.Debug("EventClientClose %+v %+v", ipc, data)
		uid := ipc.Sender
		player := getUser(uid, false)
		if player.room != nil {
			player.quitTimer = skeleton.AfterFunc(time.Second*70, func() {
				if player.room != nil {
					player.room.quit(player, QuitRoomTypeAuto)
				}
				delCacheUser(player.BaseInfo.ID)

			})
		}
	})
	conn.SubscribeEvent(common.ItemAddEvent, func(data []byte) {
		var itemReq proto.ItemReq
		if conn.DecodeClientData(data, &itemReq) {
			player := getCacheUser(itemReq.Uid)
			if player != nil {
				player.addItemFromOther(itemReq.ItemId, itemReq.Num, -int64(itemReq.Action), itemReq.Action, itemReq.ActionData1)
			}
		}
	})
	conn.SubscribeEvent(common.ItemSubEvent, func(data []byte) {
		var itemReq proto.ItemReq
		if conn.DecodeClientData(data, &itemReq) {
			player := getCacheUser(itemReq.Uid)
			if player != nil {
				player.subItemFromOther(itemReq.ItemId, itemReq.Num, -int64(itemReq.Action), itemReq.Action, itemReq.ActionData1)
			}
		}
	})
	go func() {
		for !moduleIsClose {
			now := time.Now()
			currTimeInDay = now.Day()
			currUnixTime = now.Unix()
			time.Sleep(time.Second)
		}
	}()
	roomTypes := data.GetRoomTypeConfigs()
	for _, rt := range roomTypes {
		for i := 0; i < 1; i++ {
			newRoom(0, rt)
			//time.Sleep(time.Second)
		}
	}
	err := rpc.ServerOverNats("", new(GameRPC), nil)
	log.Debug("ServerOverNats %v", err)
}

func (m *Module) OnDestroy() {
	moduleIsClose = true
	MatchRoom.Range(func(key, value interface{}) bool {
		// 踢出所有玩家
		room := value.(*Room)
		room.quitAll()
		return true
	})
	roomTypes := data.GetRoomTypeConfigs()
	for rt := range roomTypes {
		log.Debug("rt:%v", rt)
		pools, ok := data.GetAllPools(rt)
		log.Debug("pools:%+v", pools)
		if ok {
			for _, p := range pools {
				log.Debug("p:%+v", p)
				model.SavePool(p.ID, rt, p.Pool)
			}
		}

	}
	model.RWorkers.Done()
}
