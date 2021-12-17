package data

import (
	"fish_server_2021/game/data/model"
	"fish_server_2021/libs/excel"
	"lolGF/utils/econv"
	"strings"
	"sync"
)

var (
	// 怪物刷新表
	monsterGrowConfigs     map[int][]*MonsterGrowConfig // Room => Array
	monsterGrowConfigMaps  map[int]*MonsterGrowConfig   // Id => Array
	monsterGrowConfigsLock sync.RWMutex
	// 怪物阵形表
	monsterFormationConfigs map[int][]*MonsterFormationConfig // Group => Array
	// 怪物路线表
	monsterPathConfigs map[int]*MonsterPathConfig // Id => Item

	monsterConfigs     map[int]*Monster // Id => Item
	monsterConfigsLock sync.RWMutex
)

const (
	MonsterFormationTypeOne   = 1
	MonsterFormationTypeGroup = 2

	MonsterTypeSmall     = 1
	MonsterTypeLightning = 2
	MonsterTypeBlackHole = 3
	MonsterTypeRedPack   = 4
	MonsterTypeBoss      = 5
	MonsterTypePool      = 7
	MonsterTypeBomb      = 6

	CloseTypeSmall   = 1
	CloseTypeSpecial = 2
	CloseTypeRedPack = 3
	CloseTypePool    = 4
	CloseTypeBoss    = 5
)

func GetMonsterPathConfig(id int) (config *MonsterPathConfig, ok bool) {
	monsterGrowConfigsLock.RLock()
	config, ok = monsterPathConfigs[id]
	monsterGrowConfigsLock.RUnlock()
	return
}
func GetMonsterGrowConfigs(roomType int) (grows []*MonsterGrowConfig, ok bool) {
	monsterGrowConfigsLock.RLock()
	grows, ok = monsterGrowConfigs[roomType]
	monsterGrowConfigsLock.RUnlock()
	return
}
func GetFormationConfigs(group int) []*MonsterFormationConfig {
	ret, _ := monsterFormationConfigs[group]
	return ret
}

func GetMonsterConfig(id int) (*Monster, bool) {
	monsterConfigsLock.RLock()
	defer monsterConfigsLock.RUnlock()
	ret, ok := monsterConfigs[id]
	return ret, ok
}

// 怪物
type Monster struct {
	Id            int
	Name          string
	Icon          string
	Model         string //对应模型id
	Description   string
	Type          int   //怪物类型
	CloseType     int   //结算类型
	Rate          int64 //倍率
	Weight        int32 //命中权重 0=不参与命中计算
	RookieWeight  int32
	GroupID       int //命中归属组id
	BombRange     int
	BombDelay     int
	EffectScore   string
	EffectCoinNum int
	CanBoss       int
	CanBomb       int
	CanLightning  int
	CanBlackhole  int
	CanSlow       int
	DropID        int
	Proportion    float32
	DieEffect     string
	DieSound      string
	ComeOnEffect  string
	ComeOnMusic   string
	Redpack       float32
	CasinoScore   int64
}

// 刷怪规则
type MonsterGrowConfig struct {
	Id       int
	Room     int
	Type     int
	Monster  []int
	Weight   []int
	Num      int
	Interval int
	Appear   int
	Loop     int
	MaxNum   int32
	MinNum   int32
}

// 刷怪陈列配置
type MonsterFormationConfig struct {
	Id                 int
	Group              int
	Delay              float32
	Monster            int
	Path               int
	InitialOrientation int
	Move               int
	Speed              float64
	Traverse           int
	Cycles             int
	Behavior           int
	SkillId            []int
	Time               []int
	Live               int64
}

type MonsterPathConfig struct {
	Id     int
	Path   string
	Length float64
}

func initMonster() {
	monsterGrowConfigMaps = make(map[int]*MonsterGrowConfig)
	monsterGrowConfigs = make(map[int][]*MonsterGrowConfig)
	//datas := excel.LoadExcelMap("conf/excel/MonsterRefresh.xlsx")
	GetMonsterGrowByRoom(0)

	monsterFormationConfigs = make(map[int][]*MonsterFormationConfig)
	datas := excel.LoadExcelMap("conf/excel/MonsterFormation.xlsx")
	for _, data := range datas {
		tmp := &MonsterFormationConfig{}
		tmp.Id, _ = econv.Int(data["Id"])
		tmp.Group, _ = econv.Int(data["Group"])
		tmp.Delay, _ = econv.Float32(data["Delay"])
		tmp.Monster, _ = econv.Int(data["Monster"])
		tmp.Path, _ = econv.Int(data["Path"])
		tmp.InitialOrientation, _ = econv.Int(data["InitialOrientation"])
		tmp.Move, _ = econv.Int(data["Move"])
		tmp.Speed, _ = econv.Float64(data["Speed"])
		tmp.Traverse, _ = econv.Int(data["Traverse"])
		tmp.Cycles, _ = econv.Int(data["Cycles"])
		tmp.Behavior, _ = econv.Int(data["Behavior"])
		skills := strings.Split(",", data["SkillId"])
		skillLen := len(skills)
		tmp.SkillId = make([]int, skillLen)
		for i, sid := range skills {
			tmp.SkillId[i], _ = econv.Int(sid)
		}

		times := strings.Split(",", data["Time"])
		l := len(times)
		if skillLen > l {
			l = skillLen
		}
		tmp.Time = make([]int, l)
		for i, sid := range times {
			tmp.Time[i], _ = econv.Int(sid)
		}

		tmp.Live, _ = econv.Int64(data["Live"])

		_, ok := monsterFormationConfigs[tmp.Group]
		if !ok {
			monsterFormationConfigs[tmp.Group] = make([]*MonsterFormationConfig, 0, 20)
		}
		monsterFormationConfigs[tmp.Group] = append(monsterFormationConfigs[tmp.Group], tmp)
	}

	monsterPathConfigs = make(map[int]*MonsterPathConfig)
	datas = excel.LoadExcelMap("conf/excel/MonsterPath.xlsx")
	for _, data := range datas {
		tmp := &MonsterPathConfig{}
		tmp.Id, _ = econv.Int(data["Id"])
		tmp.Path = data["Path"]
		tmp.Length, _ = econv.Float64(data["Length"])

		monsterPathConfigs[tmp.Id] = tmp
	}

	monsterConfigs = make(map[int]*Monster)
	datas = excel.LoadExcelMap("conf/excel/Monster.xlsx")
	for _, data := range datas {
		tmp := &Monster{}
		tmp.Id, _ = econv.Int(data["Id"])
		tmp.Name = data["Name"]
		tmp.Icon = data["Icon"]
		tmp.Model = data["Model"]
		tmp.Description = data["Name"]
		tmp.Type, _ = econv.Int(data["Type"])
		tmp.CloseType, _ = econv.Int(data["CloseType"])
		tmp.Rate, _ = econv.Int64(data["Rate"])
		tmp.Weight, _ = econv.Int32(data["Weight"])
		tmp.RookieWeight, _ = econv.Int32(data["RookieWeight"])
		tmp.GroupID, _ = econv.Int(data["GroupID"])
		tmp.BombRange, _ = econv.Int(data["BombRange"])
		tmp.BombDelay, _ = econv.Int(data["BombDelay"])
		tmp.EffectScore = data["EffectScore"]
		tmp.EffectCoinNum, _ = econv.Int(data["EffectCoinNum"])
		tmp.CanBoss, _ = econv.Int(data["CanBoss"])
		tmp.CanBomb, _ = econv.Int(data["CanBomb"])
		tmp.CanLightning, _ = econv.Int(data["CanLightning"])
		tmp.CanBlackhole, _ = econv.Int(data["CanBlackhole"])
		tmp.CanSlow, _ = econv.Int(data["CanSlow"])
		tmp.DropID, _ = econv.Int(data["Drop"])
		tmp.Proportion, _ = econv.Float32(data["Proportion"])
		tmp.DieEffect = data["DieEffect"]
		tmp.DieSound = data["DieSound"]
		tmp.ComeOnEffect = data["ComeOnEffect"]
		tmp.ComeOnMusic = data["ComeOnMusic"]

		monsterConfigs[tmp.Id] = tmp

		//database.AdminDB.Create(tmp)
	}
	GetMonsterToMap()
}

func GetMonsterGrowByRoom(roomType int) bool {
	oldId := ""
	if roomType > 0 {
		for _, m := range monsterGrowConfigs[roomType] {
			instr, _ := econv.String(m.Id)
			oldId += "." + instr
		}
		delete(monsterGrowConfigs, roomType)
	}
	dbDatas := model.GetMonsterGrowByRoom(roomType)
	newId := ""
	for _, data := range dbDatas {
		instr, _ := econv.String(data.Id)
		newId += "." + instr

		tmp := MonsterGrowConfig{}
		tmp.Id = data.Id
		tmp.Room = data.Room
		tmp.Type = data.Type

		monster := strings.Split(data.Monster, ",")
		tmp.Monster = make([]int, len(monster))
		for i, m := range monster {
			tmp.Monster[i], _ = econv.Int(m)
		}

		weight := strings.Split(data.Weight, ",")
		tmp.Weight = make([]int, len(weight))
		for i, m := range weight {
			tmp.Weight[i], _ = econv.Int(m)
		}

		tmp.Num = data.Num
		tmp.Interval = data.Interval
		tmp.Appear = data.Appear
		tmp.Loop = data.Loop
		tmp.MaxNum = data.MaxNum
		tmp.MinNum = data.MinNum

		_, ok := monsterGrowConfigMaps[tmp.Id]
		if ok {
			*monsterGrowConfigMaps[tmp.Id] = tmp
		} else {
			monsterGrowConfigMaps[tmp.Id] = &tmp
		}

		_, ok = monsterGrowConfigs[tmp.Room]
		if !ok {
			monsterGrowConfigs[tmp.Room] = make([]*MonsterGrowConfig, 0, 20)
		}
		monsterGrowConfigs[tmp.Room] = append(monsterGrowConfigs[tmp.Room], monsterGrowConfigMaps[tmp.Id])
	}
	return oldId != newId
}

func GetMonsterToMap() {
	list := GetAllMonster()
	for _, monster := range list {
		*monsterConfigs[monster.Id] = monster
	}
}
