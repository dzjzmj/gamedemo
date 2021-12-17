package internal

import (
	"fish_server_2021/common"
	"fish_server_2021/common/proto"
	"fish_server_2021/game/data"
	"lolGF/conn"
	"lolGF/utils/econv"
)

// 客户端Key
const (

	// 其它常量
	ChangePowerAdd = 1
	ChangePowerDec = 2
	ChangePowerTop = 3
)

var (
	ClientCMD = map[int16]string{
		conn.CMConnect:         "",
		common.CMCreateRoom:    "创建房间",
		common.CMJoinRoom:      "加入房间",
		common.CMQuitRoom:      "离开房间",
		common.CMShoot:         "射击",
		common.CMHit:           "击中",
		common.CMChangePower:   "调整火力",
		common.CMHugeArmy:      "大军来袭通知",
		common.CMDeadAffect:    "死亡影响",
		common.CMReadPackAward: "获取红包奖励",
		common.CMUpPower:       "火力升级",
		common.CMSKillSwitch:   "技能开关",
		common.CMGetUserInfo:   "用户信息",
		common.CMLevelUpAward:  "用户升级领取",
		common.CMPlayerInfo:    "玩家信息",
		common.CMGetTaskAward:  "领取任务奖励",
		common.CMNewGuide:      "完成新手引导",

		common.CMGetPool:    "测试用于查看水池",
		common.CMPlayerPool: "测试用于查看水池",
		common.CMRecharge:   "充值",
	}
)

// 服务端返回Key
const (
	SMCreateRoom    = 1102 //创建房间
	SMJoinRoom      = 1103 //自己加入房间结果
	SMOtherJoinRoom = 1105 //新玩家加入房间通知
	SMQuitRoom      = 1104 //离开房间

	SMShoot              = 1110
	SMHit                = 1111
	SMChangePower        = 1112
	SMDeadAffect         = 1113
	SMHugeArmy           = 1120
	SMGrowMaster         = 1130 // 刷怪信息
	SMGrowMasters        = 1131 // 刷怪信息
	SMBuff               = 1132 // buff通知
	SMSkill              = 1133 // 技能通知
	SMDropItem           = 1140
	SMReadPackAward      = 1141
	SMReadPackResult     = 1143
	SMReadPackBulletNum  = 1144
	SMRedPackAwardConfig = 1142

	SMUpPowerNotice = 1150
	SMUpPower       = 1151
	SMSKillSwitch   = 1152

	SMLevelUp      = 1160
	SMLevelUpAward = 1161
)

const RetSuccess = 1
const RetFail = 2

type SkillSwitchReq struct {
	Skill int // 1穿透 2瞄准  3分身 4狂暴
}
type SkillStatus struct {
	Skill  int // 1穿透 2瞄准  3分身 4狂暴
	UID    int32
	Sec    int // 1开 2关
	CD     int
	Ret    int
	Start  int64
	Used   int64
	isOpen bool
	isCool bool
}

type RedPackRet struct {
	Ret int
	Id  int
}

type UpPowerRet struct {
	Config   *data.FirePower
	Ok       bool // true表示可以升级
	MaxPower int64
}
type DropItemRet struct {
	MID   int64
	UID   int32
	ID    int32
	Num   int64
	Total int64
}
type HugeArmyRet struct {
	NextMap int
	Type    int
}

type ChangePowerReq struct {
	CT int // 1 增加 2 减少
}
type ChangePowerRet struct {
	Ret   int // 1 成功 2 失败
	UID   int32
	Power int64
}

type TestRet struct {
	UID int32
	CMD int16
}
type JoinRoomReq struct {
	RT int
}
type QuitRoomRet struct {
	Ret  int
	UID  int32
	Seat int
	Type int //1主动退出 2 超时退出
}

type UserInfoRet struct {
	ID        int32
	Name      string
	Avatar    string
	UserLevel int
	Coin      int64 // 金币
	Pearl     int64 // 珍珠
	Gem       int64 // 宝石
	Power     int64 // 当前火力
	Hero      int32 // 英雄ID
	Exp       int64 // 经验
	Items     proto.ItemsResp
}
type JoinRoomRet struct {
	Ret   int
	ID    int32
	User  map[int]UserInfoRet // 房间玩家信息 位置 => 玩家
	MapID int                 // 地图ID
	Seat  int
}
type JoinRoomOtherRet struct {
	Ret  int
	Seat int
	User UserInfoRet // 房间玩家信息 位置 => 玩家
}
type ShootRet struct {
	ID  int64 //子弹ID
	Ret int
}
type ShootWrap struct {
	UID int32

	ID     int64 //子弹ID
	Coin   int64 //最新金币
	MID    int64
	Vector Xyz
}
type Xyz struct {
	X float32
	Y float32
	Z float32
}

func (resp *ShootWrap) decode(data interface{}) bool {
	var tmp []interface{}
	if !conn.DecodeClientData(data, &tmp) || len(tmp) < 3 {
		return false
	}

	var ok bool
	resp.ID, ok = econv.Int64(tmp[0])
	if !ok {
		return false
	}
	resp.MID, ok = econv.Int64(tmp[1])
	if !ok {
		return false
	}

	vector, ok := tmp[2].([]interface{})
	if !ok {
		return false
	}
	resp.Vector.X, _ = econv.Float32(vector[0])
	resp.Vector.Y, _ = econv.Float32(vector[1])
	resp.Vector.Z, _ = econv.Float32(vector[2])

	return true
}

func (resp *ShootWrap) encode() []interface{} {
	tmp := make([]interface{}, 0, 6)

	tmp = append(tmp, resp.UID)
	tmp = append(tmp, resp.ID)

	tmp = append(tmp, resp.MID)
	tmp = append(tmp, [3]float32{resp.Vector.X, resp.Vector.Y, resp.Vector.Z})

	tmp = append(tmp, resp.Coin)

	return tmp
}

type HitWrap struct {
	UID  int32
	ID   int64 //子弹ID
	Coin int64 //最新金币

	Num       int
	MID       []int64
	Win       []int64
	AddItemId int
}

func (resp *HitWrap) decode(data interface{}) bool {
	var tmp []int64
	if !conn.DecodeClientData(data, &tmp) || len(tmp) < 2 {
		return false
	}
	for _, v := range tmp {
		if v < 0 {
			return false
		}
	}
	resp.ID = tmp[0]
	resp.Num = int(tmp[1])
	if resp.Num > 100 || len(tmp) < 2+resp.Num {
		return false
	}

	if resp.Num > 0 {
		resp.MID = make([]int64, resp.Num)
		for ii := 0; ii < resp.Num; ii++ {
			resp.MID[ii] = tmp[2+ii]
		}
	}

	return true
}

func (resp *HitWrap) encode() []interface{} {
	tmp := make([]interface{}, 0, 8+2*len(resp.MID))

	tmp = append(tmp, resp.UID)
	tmp = append(tmp, resp.ID)
	tmp = append(tmp, resp.Coin)

	tmp = append(tmp, len(resp.MID))
	for _, fid := range resp.MID {
		tmp = append(tmp, fid)
	}
	for _, win := range resp.Win {
		tmp = append(tmp, win)
	}
	//tmp = append(tmp, resp.AddItemId)

	return tmp
}

type DeadAffectWrap struct {
	UID    int32
	ID     int64 //子弹ID
	DeadId int64
	Coin   int64 //最新金币

	Num int
	MID []int64
	Win []int64
}

func (resp *DeadAffectWrap) decode(data interface{}) bool {
	var tmp []int64
	if !conn.DecodeClientData(data, &tmp) || len(tmp) < 2 {
		return false
	}
	for _, v := range tmp {
		if v < 0 {
			return false
		}
	}
	resp.ID = tmp[0]
	resp.DeadId = tmp[1]
	resp.Num = int(tmp[2])
	mStartKey := 3
	if resp.Num > 100 || len(tmp) < mStartKey+resp.Num {
		return false
	}

	if resp.Num > 0 {
		resp.MID = make([]int64, resp.Num)
		for ii := 0; ii < resp.Num; ii++ {
			resp.MID[ii] = tmp[mStartKey+ii]
		}
	}

	return true
}

func (resp *DeadAffectWrap) encode() []interface{} {
	tmp := make([]interface{}, 0, 8+2*len(resp.MID))

	tmp = append(tmp, resp.UID)
	tmp = append(tmp, resp.ID)
	tmp = append(tmp, resp.DeadId)
	tmp = append(tmp, resp.Coin)

	tmp = append(tmp, len(resp.MID))
	for _, fid := range resp.MID {
		tmp = append(tmp, fid)
	}
	for _, win := range resp.Win {
		tmp = append(tmp, win)
	}

	return tmp
}
