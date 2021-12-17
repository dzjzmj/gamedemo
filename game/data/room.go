package data

import (
	"fish_server_2021/game/data/model"
	"fish_server_2021/libs/excel"
	"lolGF/utils/econv"
	"strings"
	"sync"
)

var roomTypeConfigs map[int]*RoomType
var mapConfigs map[int]*Map
var mapConfigsLock sync.RWMutex

const RoomTypeRedPack = 1
const RoomTypeCasino = 2

type RoomType struct {
	Id             int
	Name           string
	RoomType       int
	IncludePower   []int
	Describe       string
	MapID          []int
	NeedVip        int
	NeedMoney      int64
	NeedPower      int64
	MonsterRevenue float64
	RedpackRevenue float64
	SpecialRevenue float64
	BossRevenue    float64
	ChestRevenue   float64
}
type Map struct {
	Id           int
	Name         string
	InvadeTime   int
	InvadeGroup  []int
	Weight       []int
	IntervalTime []int
	Res          string
	BgMusicID    int
}

func InitRoomData() {
	initPool()
	initBulletRate()
	initHero()
	initMonster()
	InitLevelData()
	InitRadPack()
	mapConfigs = make(map[int]*Map)
	datas := excel.LoadExcelMap("conf/excel/MapConfig.xlsx")
	for _, data := range datas {
		tmp := &Map{}
		tmp.Id, _ = econv.Int(data["Id"])
		tmp.Name = data["Name"]
		tmp.InvadeTime, _ = econv.Int(data["InvadeTime"])
		groups := strings.Split(data["InvadeGroup"], ";")
		tmp.InvadeGroup = make([]int, len(groups))
		for i, str := range groups {
			tmp.InvadeGroup[i], _ = econv.Int(str)
		}

		weights := strings.Split(data["Weight"], ";")
		tmp.Weight = make([]int, len(groups))
		for i, str := range weights {
			tmp.Weight[i], _ = econv.Int(str)
		}

		times := strings.Split(data["IntervalTime"], ";")
		tmp.IntervalTime = make([]int, len(groups))
		for i, str := range times {
			tmp.IntervalTime[i], _ = econv.Int(str)
		}

		tmp.Res = data["Res"]
		tmp.BgMusicID, _ = econv.Int(data["BgMusicID"])

		mapConfigs[tmp.Id] = tmp

	}

	bulletRateConfigs = make(map[int][]*BulletRate)

	roomTypeConfigs = make(map[int]*RoomType)
	poolsConfig = make(map[int]map[int]*Pools) // Room => Map[Id => Pools]
	dataRooms := excel.LoadExcelMap("conf/excel/Room.xlsx")
	for _, data := range dataRooms {
		tmp := &RoomType{}
		tmp.Id, _ = econv.Int(data["Id"])

		poolsConfig[tmp.Id] = make(map[int]*Pools)
		for _, pool := range poolsMapConfig {
			newPool := pool
			dbPool := model.GetPool(newPool.ID, tmp.Id)
			newPool.Pool = dbPool.Pool
			poolsConfig[tmp.Id][pool.ID] = &newPool
		}

		tmp.Name = data["Name"]
		tmp.RoomType, _ = econv.Int(data["RoomType"])

		powers := strings.Split(data["IncludePower"], ";")
		tmp.IncludePower = make([]int, len(powers))
		for i, str := range powers {
			tmp.IncludePower[i], _ = econv.Int(str)
		}

		tmp.Describe = data["Describe"]

		maps := strings.Split(data["MapId"], ";")
		tmp.MapID = make([]int, len(maps))
		for i, str := range maps {
			tmp.MapID[i], _ = econv.Int(str)
		}

		tmp.NeedVip, _ = econv.Int(data["NeedVip"])
		tmp.NeedMoney, _ = econv.Int64(data["NeedMoney"])
		tmp.NeedPower, _ = econv.Int64(data["NeedPower"])
		tmp.MonsterRevenue, _ = econv.Float64(data["MonsterRevenue"])
		tmp.RedpackRevenue, _ = econv.Float64(data["RedpackRevenue"])
		tmp.SpecialRevenue, _ = econv.Float64(data["SpecialRevenue"])
		tmp.BossRevenue, _ = econv.Float64(data["BossRevenue"])
		tmp.ChestRevenue, _ = econv.Float64(data["ChestRevenue"])

		roomTypeConfigs[tmp.Id] = tmp

		bulletRateConfigs[tmp.Id] = make([]*BulletRate, 0, len(tmp.IncludePower))
		for _, pid := range tmp.IncludePower {
			t, ok := bulletRateMapConfigs[pid]
			if ok {
				bulletRateConfigs[tmp.Id] = append(bulletRateConfigs[tmp.Id], t)
			}
		}

	}
	GetRoomTypeToMap()
}

func GetRoomTypeToMap() {
	list := model.GetAllRoomType()
	for _, room := range list {
		tmp := RoomType{
			Id:             room.Id,
			Name:           room.Name,
			RoomType:       room.RoomType,
			IncludePower:   nil,
			Describe:       room.Describe,
			MapID:          nil,
			NeedVip:        room.NeedVip,
			NeedMoney:      room.NeedMoney,
			NeedPower:      room.NeedPower,
			MonsterRevenue: room.MonsterRevenue,
			RedpackRevenue: room.RedpackRevenue,
			SpecialRevenue: room.SpecialRevenue,
			BossRevenue:    room.BossRevenue,
			ChestRevenue:   room.ChestRevenue,
		}
		_, ok := poolsConfig[tmp.Id]
		if !ok {
			poolsConfig[tmp.Id] = make(map[int]*Pools)
			for _, pool := range poolsMapConfig {
				newPool := pool
				dbPool := model.GetPool(newPool.ID, tmp.Id)
				newPool.Pool = dbPool.Pool
				poolsConfig[tmp.Id][pool.ID] = &newPool
			}
		}
		powers := strings.Split(room.IncludePower, ";")
		tmp.IncludePower = make([]int, len(powers))
		for i, str := range powers {
			tmp.IncludePower[i], _ = econv.Int(str)
		}

		maps := strings.Split(room.MapID, ";")
		tmp.MapID = make([]int, len(maps))
		for i, str := range maps {
			tmp.MapID[i], _ = econv.Int(str)
		}
		_, ok = roomTypeConfigs[tmp.Id]
		if ok {
			*roomTypeConfigs[tmp.Id] = tmp
		} else {
			roomTypeConfigs[tmp.Id] = &tmp
		}
		bulletRateConfigs[tmp.Id] = make([]*BulletRate, 0, len(tmp.IncludePower))
		for _, pid := range tmp.IncludePower {
			t, ok := bulletRateMapConfigs[pid]
			if ok {
				bulletRateConfigs[tmp.Id] = append(bulletRateConfigs[tmp.Id], t)
			}
		}
	}
}

func GetRoomTypeConfigs() map[int]*RoomType {
	return roomTypeConfigs
}
func GetRoomTypeConfig(id int) (*RoomType, bool) {
	mapConfigsLock.RLock()
	config, ok := roomTypeConfigs[id]
	mapConfigsLock.RUnlock()
	return config, ok
}
func GetMap(id int) (*Map, bool) {
	mapConfigsLock.RLock()
	m, ok := mapConfigs[id]
	mapConfigsLock.RUnlock()
	return m, ok
}
