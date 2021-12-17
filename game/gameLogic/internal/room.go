package internal

import (
	"fish_server_2021/common"
	"fish_server_2021/common/proto"
	"fish_server_2021/game/base"
	"fish_server_2021/game/data"
	json "github.com/json-iterator/go"
	"lolGF/conn"
	"lolGF/libs/msgpack"
	"lolGF/log"
	"lolGF/module"
	"lolGF/utils/emath"
	"sync"
	"time"
)

const (
	RoomStateInit = iota
	RoomStateReady

	RoomMaxPlayerNum    = 2
	quitTimeoutSec      = 3600 // 自动踢出时间 需求 1分钟
	quitRoomCacheSec    = 300  // 进入离开前房间缓存时间  需求为5分秒
	QuitRoomTypePlayer  = 1
	QuitRoomTypeAuto    = 2
	QuitRoomTypeDestroy = 3
)

var MatchRoom sync.Map

type Room struct {
	ID           int32
	MaxPlayerNum int //最大玩家数量
	RoomType     *data.RoomType
	OwnerId      int32
	State        int8
	seats        map[int]*Player
	seatUserIds  []int32
	Map          *data.Map // 初始化时随机一个mapId
	ArmyTimer    *module.Timer

	MonsterGrow      []*MonsterGrowConfig
	monsters         map[int64]*Monster
	configMonsterNum map[int]int32
	//perMonsterIndex      map[int][]int //上次生成的怪
	//perMonsterIndexLock  sync.RWMutex
	configMonsterNumLock sync.RWMutex

	monsterCurrId int64

	bulletRates []*data.BulletRate

	updateTime int64

	isGrowHugeArmy bool
}

func (room *Room) init() {
	// 随机地图
	room.changeMap()
	//room.perMonsterIndex = make(map[int][]int)
	// 初始怪物数据
	room.monsters = make(map[int64]*Monster)
	room.configMonsterNum = make(map[int]int32)

	room.initGrowMonster()

	skeleton.GoSafe(func() {
		for !moduleIsClose {
			room.monsterDie()
			time.Sleep(time.Second)
		}
	})
	r := emath.RandInt(1, 16)
	skeleton.AfterFunc(time.Duration(r)*time.Second, func() {
		room.State = RoomStateReady
	})

	room.seatUserIds = make([]int32, room.MaxPlayerNum)

	room.bulletRates, _ = data.GetBulletRateConfigs(room.RoomType.Id)
	//fmt.Printf("bulletRates %v %+v", room.RoomType.Id, room.bulletRates)
	room.timeoutPlayers()
}
func (room *Room) initGrowMonster() {
	// 停止原来的刷怪
	if room.ArmyTimer != nil {
		room.ArmyTimer.Stop()
	}
	for _, config := range room.MonsterGrow {
		if config.Timer != nil {
			config.Timer.Stop()
		}
	}

	// 重置新的
	room.MonsterGrow = make([]*MonsterGrowConfig, 0, 20)
	growConfig, ok := data.GetMonsterGrowConfigs(room.RoomType.Id)
	if ok {
		for _, config := range growConfig {
			growConfigNew := MonsterGrowConfig{}
			growConfigNew.Config = config
			room.MonsterGrow = append(room.MonsterGrow, &growConfigNew)
		}
	}

	room.isGrowHugeArmy = false
	room.growMonsters()
}
func (room *Room) changeMap() *data.Map {
	mapNum := len(room.RoomType.MapID)
	//fmt.Printf("MapID %+v\n", room.RoomType.MapID)
	if mapNum < 1 {
		return nil
	}
	var mid int
	var mapIndex int
	if mapNum == 1 {
		mid = room.RoomType.MapID[0]
		goto getMapLabel
	}
randLabel:
	mapIndex = emath.RandInt(0, mapNum)
	// 随机地图
	mid = room.RoomType.MapID[mapIndex]
	if room.Map != nil && room.Map.Id == mid {
		goto randLabel
	}
getMapLabel:
	m, _ := data.GetMap(mid)

	room.Map = m
	return m
}

func (room *Room) timeoutPlayers() {
	skeleton.AfterFunc(time.Duration(quitTimeoutSec)*time.Second, func() {
		for _, player := range room.seats {
			if currUnixTime-player.playerCount.UpdateTime > quitTimeoutSec {
				room.quit(player, QuitRoomTypeAuto)
			}
		}
		skeleton.GoSafe(func() {
			room.timeoutPlayers()
		})
	})
}

func (room *Room) allUserIds() []int32 {
	uids := make([]int32, 0, 6)
	for _, uid := range room.seatUserIds {
		if uid > 0 {
			uids = append(uids, uid)
		}
	}
	return uids
	//return room.seatUserIds
}

func (room *Room) addPlayer(player *Player, retFail bool) bool {
	if player == nil {
		return false
	}
	log.Debug("player %+v", player.BaseInfo)
	if player.BaseInfo.Coin < room.RoomType.NeedMoney {
		base.SendRetToGate(player.BaseInfo.ID, SMJoinRoom, 3)
		return false
	}
	if player.BaseInfo.MaxPower < room.RoomType.NeedPower {
		base.SendRetToGate(player.BaseInfo.ID, SMJoinRoom, 5)
		return false
	}
	player.playerCount.UpdateTime = currUnixTime
	emptySeat := -1
	var ps common.PlayStatus
	var dd []byte
	if player.room != nil {
		if player.room.ID == room.ID {
			goto inRoomLabel
		}
	}
	for seat, uid := range room.seatUserIds {
		if uid == 0 {
			emptySeat = seat
			room.seatUserIds[seat] = player.BaseInfo.ID
			room.seats[seat] = player
			break
		}
	}
	if emptySeat == -1 {
		if retFail {
			base.SendRetToGate(player.BaseInfo.ID, SMJoinRoom, 2)
		}
		return false
	}

	ps = common.PlayStatus{
		UID:    player.BaseInfo.ID,
		Status: "game",
		RT:     room.RoomType.Id,
	}
	dd, _ = msgpack.Marshal(ps)
	_ = conn.SendServerEvent(common.PlayStatusChangeEvent, dd)

	player.room = room
	player.Seat = emptySeat

inRoomLabel:
	for i, t := range room.bulletRates {
		if t.Gun == player.BaseInfo.Power {
			player.currentBulletRate = i
		}
	}

	// 返回加入成功信息
	userRoomMap.Store(player.BaseInfo.ID, room)
	player.setBulletRates()
	room.joinRoomRetMsg(player)
	common.TaskGetRunObjectiveType(player.BaseInfo.ID)
	skeleton.AfterFunc(time.Millisecond*100, func() {
		room.sendMonstersRet(player.BaseInfo.ID)
		player.upPowerNotice(true)

		if player.BaseInfo.LoginDay <= 1 {
			if player.BaseInfo.NewGuide == 0 {
				base.SendMsgToGate(player.BaseInfo.ID, common.SMNewGuide, 1)
			}
		}

		for _, p := range room.seats {
			// 技能
			for _, skill := range p.skillConfig {
				if skill.Start > 0 {
					skill.Used = time.Now().UnixNano()/1000/1000 - skill.Start
					room.notifyAll(SMSKillSwitch, skill)
				}
			}
		}
	})

	return true
}

var delQuitTimer *module.Timer

func (room *Room) notifyAll(cmd int16, data interface{}) {
	uids := room.allUserIds()
	if len(uids) > 0 {
		conn.SendMsgToClients(uids, cmd, data, false)
	}
}

func (room *Room) quitAll() {
	for _, player := range room.seats {
		room.quit(player, QuitRoomTypeDestroy)
	}
}

func (room *Room) quit(player *Player, quitType int) {
	ps := common.PlayStatus{
		UID:    player.BaseInfo.ID,
		Status: "quitGame",
		RT:     room.RoomType.Id,
	}
	dd, _ := msgpack.Marshal(ps)
	_ = conn.SendServerEvent(common.PlayStatusChangeEvent, dd)

	seat := player.Seat

	player.room = nil
	player.Seat = -1
	ret := QuitRoomRet{
		Ret:  1,
		UID:  player.BaseInfo.ID,
		Type: quitType,
		Seat: seat,
	}
	userRoomMap.Delete(player.BaseInfo.ID)
	player.playerCount.UpdateTime = currUnixTime
	quitRoomMap.Store(player.BaseInfo.ID, room)
	// TODO 同步金币 战斗记录等数据

	room.notifyAll(SMQuitRoom, ret)
	room.seatUserIds[seat] = 0
	player.bulletTimeout()
	player.SyncCoin()

	delete(room.seats, seat)

	if delQuitTimer == nil {
		delQuitTimer = skeleton.AfterFunc(time.Minute*5, func() {
			delQuitRoom()
		})
	}
	if quitType == QuitRoomTypeDestroy {
		player.saveItemData()
	} else {
		skeleton.GoSafe(func() {
			player.saveItemData()
		})
	}
}
func (player *Player) saveItemData() {
	rate := player.bulletRates[player.currentBulletRate]
	// 保存当前火力
	itemDatas := make(map[int32]int64)
	itemDatas[common.PowerItemId] = rate.Gun
	itemDatas[common.RedPackBulletNum] = player.BaseInfo.BulletNum
	itemDatas[common.ExpItemId] = player.BaseInfo.Exp

	for itemId, num := range itemDatas {
		_ = common.ItemClient.SetItem(proto.ItemReq{
			Uid:     player.BaseInfo.ID,
			ItemId:  itemId,
			Num:     num,
			Action:  1,
			Service: "game",
		})
	}

	player.playerCount.DropItem, _ = json.MarshalToString(player.dropItemNum)
	_ = player.playerCount.Save()
}
func (room *Room) joinRoomRetMsg(joinPlayer *Player) {
	users := make(map[int]UserInfoRet)

	otherUsersRet := JoinRoomOtherRet{
		Ret: 1,
	}
	otherUserIds := make([]int32, 0, 6)
	for seat, player := range room.seats {
		var rate *data.BulletRate
		if player.currentBulletRate > len(player.bulletRates)-1 {
			rate = &data.BulletRate{}
			log.Error("%v %v", player.BaseInfo, player.bulletRates)
		} else {
			rate = player.bulletRates[player.currentBulletRate]
		}

		user := UserInfoRet{
			ID:        player.BaseInfo.ID,
			Name:      player.BaseInfo.Name,
			Avatar:    player.BaseInfo.Avatar,
			UserLevel: player.BaseInfo.UserLevel,
			Coin:      player.BaseInfo.Coin,
			Pearl:     player.BaseInfo.Pearl,
			Gem:       player.BaseInfo.Gem,
			Exp:       player.BaseInfo.Exp,
			Power:     rate.Gun,
			Hero:      player.BaseInfo.Hero,
		}
		if user.ID == joinPlayer.BaseInfo.ID {
			user.Items = joinPlayer.BaseInfo.Items
			otherUsersRet.User = user
			otherUsersRet.Seat = seat
		} else {
			otherUserIds = append(otherUserIds, user.ID)
			user.Items = make(proto.ItemsResp)
		}
		users[seat] = user

	}

	ret := JoinRoomRet{
		Seat:  joinPlayer.Seat,
		Ret:   1,
		ID:    room.ID,
		User:  users,
		MapID: room.Map.Id,
	}
	base.SendMsgToGate(joinPlayer.BaseInfo.ID, SMJoinRoom, ret)
	if len(otherUserIds) > 0 {
		conn.SendMsgToClients(otherUserIds, SMOtherJoinRoom, otherUsersRet, false)
	}
	skeleton.AfterFunc(time.Millisecond*100, func() {
		joinPlayer.checkRedPackFinish(true)
		joinPlayer.sendRedPackAwardConfig()
	})

}
