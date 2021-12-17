package internal

import (
	"fish_server_2021/game/data"
	"lolGF/log"
	"math/rand"
)

func getOldRoom(player *Player) *Room {
	return player.room
}

func joinRoom(player *Player, req *JoinRoomReq) *Room {
	oldRoom := getOldRoom(player)
	var room *Room

	if oldRoom != nil {
		room = oldRoom
		// 已经在房间
		room.addPlayer(player, true)
		return room
	}

	// 进入离开5分钟前的房间
	quitRoom := getQuitRoom(player)
	if quitRoom != nil && quitRoom.RoomType.Id == req.RT {
		if quitRoom.addPlayer(player, false) {
			return quitRoom
		}
	}
	flag := true
	MatchRoom.Range(func(key, value interface{}) bool {
		room = value.(*Room)
		num := len(room.seats)
		if room.RoomType.Id != req.RT {
			return true
		}
		if num >= room.MaxPlayerNum {
			// 满了
			return true
		}
		if num == 1 {
			flag = false
			// 先返回有一个人的
			return false
		}
		return true
	})
	if flag {
		MatchRoom.Range(func(key, value interface{}) bool {
			room = value.(*Room)
			if room.RoomType.Id != req.RT {
				return true
			}
			num := len(room.seats)
			if num >= room.MaxPlayerNum {
				// 满了
				return true
			}

			//if room.State == RoomStateReady {
			//	return false
			//}
			return true
		})
	}
	num := len(room.seats)
	if num >= room.MaxPlayerNum || room.RoomType.Id != req.RT {
		rooType, ok := data.GetRoomTypeConfig(req.RT)
		if ok {
			room = newRoom(player.BaseInfo.ID, rooType)
		} else {
			log.Debug("GetRoomTypeConfig !ok %v", req.RT)
			return nil
		}

	} else {
		//MatchRoom.Delete(room.ID)
		skeleton.GoSafe(func() {
			roomNum := 0
			MatchRoom.Range(func(key, value interface{}) bool {
				room := value.(*Room)
				num = len(room.seats)
				if num < room.MaxPlayerNum && req.RT == room.RoomType.Id {
					roomNum++
				}
				return true
			})
			if roomNum < 5 {
				rooType, ok := data.GetRoomTypeConfig(req.RT)
				if ok {
					newRoom(0, rooType)
				}
			}
		})

	}
	room.addPlayer(player, true)
	return room
}

func quitRoom(player *Player) {
	if player.room == nil {
		return
	}
	player.room.quit(player, QuitRoomTypePlayer)
}
func newRoom(uid int32, rt *data.RoomType) *Room {
	room := &Room{
		ID:           getNewRoomID(),
		OwnerId:      uid,
		seats:        make(map[int]*Player),
		State:        RoomStateInit,
		RoomType:     rt,
		MaxPlayerNum: RoomMaxPlayerNum,
	}
	MatchRoom.Store(room.ID, room)
	room.init()
	return room
}
func getNewRoomID() int32 {
	RoomID := rand.Int31()
	for _, ok := MatchRoom.Load(RoomID); ok; {
		RoomID = rand.Int31()
	}
	return RoomID
}
func getQuitRoom(player *Player) *Room {

	roomMap, ok := quitRoomMap.Load(player.BaseInfo.ID)
	if ok {
		room := roomMap.(*Room)
		if currUnixTime-player.playerCount.UpdateTime > quitRoomCacheSec {
			return nil
		}
		return room
	}
	return nil
}
func delQuitRoom() {

	quitRoomMap.Range(func(key, value interface{}) bool {
		room := value.(*Room)
		if currUnixTime-room.updateTime > quitRoomCacheSec {
			quitRoomMap.Delete(key)
		}
		return true
	})
	delQuitTimer = nil
}
