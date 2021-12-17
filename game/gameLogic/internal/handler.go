package internal

import (
	"fish_server_2021/common"
	"fish_server_2021/common/proto"
	"fish_server_2021/game/base"
	"lolGF/conn"
	"lolGF/log"
	"lolGF/utils/econv"
	"time"
)

// 客户端消息
func HandleClientMSG(req conn.ClientRequestInfo, data interface{}) {
	log.Debug("HandleClientMSG %+v", req)
	uid := req.UID
	cmd := req.CMD
	switch cmd {
	case conn.CMConnect:
		{
			// 用户重连逻辑
			//player := getUser(uid, false)
			//if player.quitTimer != nil {
			//	player.quitTimer.Stop()
			//}
			//if player.room != nil {
			//	isSame := time.Unix(player.playerCount.UpdateTime, 0).Day() == currTimeInDay
			//	if isSame {
			//		// 已经在房间
			//		player.room.addPlayer(player, true)
			//	}
			//}
		}
	case common.CMCreateRoom:
		ChanRPC.GoFunc(handleCreateRoom, uid, cmd)
	case common.CMJoinRoom:
		ChanRPC.GoFunc(handleJoinRoom, uid, data)
	case common.CMChangePower:
		skeleton.GoSafe(func() {
			handleChangePower(uid, data)
		})
	case common.CMQuitRoom:
		ChanRPC.GoFunc(handleQuitRoom, uid, data)
	case common.CMShoot:
		skeleton.GoSafe(func() {
			handleShoot(uid, data)
		})
	case common.CMHit:
		skeleton.GoSafe(func() {
			handleHit(uid, data)
		})
	case common.CMDeadAffect:
		skeleton.GoSafe(func() {
			handleDeadAffect(uid, data)
		})
	case common.CMReadPackAward:
		ChanRPC.GoFunc(handleReadPackAward, uid)
	case common.CMLevelUpAward:
		ChanRPC.GoFunc(handleUpLevelAward, uid)
	case common.CMUpPower:
		skeleton.GoSafe(func() {
			handleUpPower(uid)
		})

	case common.CMSKillSwitch:
		ChanRPC.GoFunc(handleSkillSwitch, uid, data)

	case common.CMPlayerInfo:
		ChanRPC.GoFunc(handlePlayerInfo, uid, data)
	case common.CMGetPool:
		skeleton.GoSafe(func() {
			testGetPool(uid)
		})
	case common.CMPlayerPool:
		skeleton.GoSafe(func() {
			testGetPlayerPool(uid)
		})

	case common.CMRecharge:
		num, _ := econv.Int64(data)
		player := getUser(uid, false)
		if player == nil {
			return
		}
		if player.room == nil {
			return
		}
		if num > 0 {
			//player.recharge(num)
		}
	case common.CMGetTaskAward:
		skeleton.AfterFunc(time.Second, func() {
			common.TaskGetRunObjectiveType(uid)
		})

	case common.CMNewGuide:
		skeleton.GoSafe(func() {
			player := getUser(uid, false)
			if player != nil {
				player.BaseInfo.NewGuide = 1
				base.SendRetToGate(uid, common.CMNewGuide, 1)
			}
		})
	}

}

type PlayerInfoRet struct {
	Ret  int
	User common.UserInfo
}

func handlePlayerInfo(args []interface{}) {
	uid := args[0].(int32)
	player := getUser(uid, false)
	if player == nil {
		return
	}
	if player.room == nil {
		return
	}
	seat, ok := econv.Int(args[1])
	if ok {
		p, ok := player.room.seats[seat]
		if ok {
			info := *p.BaseInfo
			if info.ID != uid {
				info.Items = make(proto.ItemsResp)
			}
			base.SendMsgToGate(uid, common.CMPlayerInfo, PlayerInfoRet{
				Ret:  1,
				User: info,
			})
		}

	}
}
func testGetPlayerPool(uid int32) {
	player := getUser(uid, false)
	if player == nil {
		return
	}
	if player.room == nil {
		return
	}
	player.getPool()
}
func testGetPool(uid int32) {
	player := getUser(uid, false)
	if player == nil {
		return
	}
	if player.room == nil {
		return
	}
	player.room.getPools(player)
}

func handleUpPower(uid int32) {
	player := getUser(uid, false)
	if player == nil {
		return
	}
	player.upPower()
}
func handleUpLevelAward(args []interface{}) {
	uid := args[0].(int32)
	player := getUser(uid, false)
	if player == nil {
		return
	}
	player.getUpLevelAward()
}
func handleReadPackAward(args []interface{}) {
	uid := args[0].(int32)
	player := getUser(uid, false)
	if player == nil {
		return
	}
	player.getRedPackAward()
}
func handleDeadAffect(uid int32, data interface{}) {
	player := getUser(uid, false)
	if player == nil {
		return
	}
	if player.room == nil {
		return
	}
	var req DeadAffectWrap
	if !req.decode(data) {
		base.SendRetToGate(uid, SMDeadAffect, -1)
		return
	}
	//log.Debug("DeadAffectWrap %+v", req)

	player.room.deadAffect(player, &req)
}
func handleHit(uid int32, data interface{}) {
	player := getUser(uid, false)
	if player == nil {
		return
	}
	if player.room == nil {
		return
	}
	var req HitWrap
	if !req.decode(data) {
		return
	}

	player.playerCount.UpdateTime = currUnixTime
	player.room.hit(player, &req)
}
func handleShoot(uid int32, data interface{}) {
	var req ShootWrap
	req.decode(data)
	player := getUser(uid, false)
	if player == nil {
		return
	}
	if player.room == nil {
		return
	}
	player.room.shoot(player, &req)
}
func handleChangePower(uid int32, data interface{}) {
	var req ChangePowerReq
	if !conn.DecodeClientData(data, &req) {
		base.SendRetToGate(uid, SMJoinRoom, 99)
		return
	}
	player := getUser(uid, false)
	//log.Debug("handleChangePower %v %v %+v", uid, player, req)
	if player != nil {
		player.changePower(&req)
	}

}
func handleQuitRoom(args []interface{}) {
	uid := args[0].(int32)
	player := getUser(uid, false)
	if player == nil {
		return
	}
	quitRoom(player)
}
func handleSkillSwitch(args []interface{}) {
	uid := args[0].(int32)
	data := args[1]
	player := getUser(uid, false)
	if player == nil {
		return
	}
	var req SkillSwitchReq
	if !conn.DecodeClientData(data, &req) {
		base.SendRetToGate(uid, SMSKillSwitch, -1)
		return
	}
	player.skillSwitch(&req)
}
func handleJoinRoom(args []interface{}) {
	uid := args[0].(int32)
	var req JoinRoomReq
	if !conn.DecodeClientData(args[1], &req) {
		base.SendRetToGate(uid, SMJoinRoom, 99)
		return
	}

	player := getUser(uid, true)
	joinRoom(player, &req)
}

func handleCreateRoom(args []interface{}) {
	uid := args[0].(int32)
	cmd := args[1].(int16)

	ret := TestRet{
		UID: uid,
		CMD: cmd,
	}
	base.SendMsgToGateEx(uid, SMCreateRoom, ret, true)

}
