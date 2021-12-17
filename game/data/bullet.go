package data

import (
	"fish_server_2021/libs/excel"
	"lolGF/utils/econv"
	"strings"
	"sync"
)

var bulletRateConfigs map[int][]*BulletRate  // Room => Array
var bulletRateMapConfigs map[int]*BulletRate // ID => Array
var bulletMapConfigs map[int]*Bullet         // ID => Array
var firePowerConfigs []*FirePower
var bulletRateConfigsLock sync.RWMutex

func GetBulletConfigs(id int) (configs *Bullet, ok bool) {
	bulletRateConfigsLock.RLock()
	configs, ok = bulletMapConfigs[id]
	bulletRateConfigsLock.RUnlock()
	return
}

func GetBulletRateConfigs(roomType int) (configs []*BulletRate, ok bool) {
	bulletRateConfigsLock.RLock()
	configs, ok = bulletRateConfigs[roomType]
	bulletRateConfigsLock.RUnlock()
	return
}

func GetFirePowerConfigs() (configs []*FirePower) {
	return firePowerConfigs
}

type Bullet struct {
	Id          int
	BuffID      int
	Probability int
	HP          int
}
type BulletRate struct {
	Id          int
	Gun         int64
	Room        int
	Redpack     float32
	MonsterRate int32
	RedpackRate int32
	SpecialRate int32
	BoosRate    int32
	ChestRate   int32
	MonsterPool int
	RedpackPool int
	SpecialPool int
	BossPool    int
	ChestPool   int
}

type FirePower struct {
	Id   int
	Name string // 进度名字
	//Order              int
	Gem        int64 //所需消耗宝石数量
	Power      int64 //激活火力
	Award      []AwardItem
	Unfinished string //任务中的文本描述
	Finished   string //任务完成的文本描述
}

func initBulletRate() {
	firePowerConfigs = make([]*FirePower, 0, 30)
	filePowerData := excel.LoadExcelMap("conf/excel/FirePower.xlsx")
	for _, data := range filePowerData {
		tmp := &FirePower{}
		tmp.Id, _ = econv.Int(data["Id"])
		tmp.Name = data["Name"]
		//tmp.Order, _ = econv.Int(data["Order"])
		tmp.Gem, _ = econv.Int64(data["AddEnergy"])
		tmp.Power, _ = econv.Int64(data["ActivateBullet"])

		awards := strings.Split(data["Award"], ";")
		tmp.Award = make([]AwardItem, len(awards))
		for i, str := range awards {
			awardItems := strings.Split(str, ",")

			//class, _ := econv.Int(awardItems[0])
			itemId, _ := econv.Int32(awardItems[0])
			num, _ := econv.Int64(awardItems[1])

			item := AwardItem{
				Id:  itemId,
				Num: num,
				//Class: class,
			}
			tmp.Award[i] = item
		}

		tmp.Unfinished = data["DescribeUnfinished"]
		tmp.Finished = data["DescribeFinished"]

		firePowerConfigs = append(firePowerConfigs, tmp)
	}

	bulletRateMapConfigs = make(map[int]*BulletRate)
	datas := excel.LoadExcelMap("conf/excel/BulletRate.xlsx")
	for _, data := range datas {
		tmp := &BulletRate{}
		tmp.Id, _ = econv.Int(data["Id"])
		tmp.Gun, _ = econv.Int64(data["Gun"])
		tmp.Room, _ = econv.Int(data["Room"])
		tmp.Redpack, _ = econv.Float32(data["Redpack"])
		tmp.MonsterRate, _ = econv.Int32(data["MonsterRate"])
		tmp.RedpackRate, _ = econv.Int32(data["RedpackRate"])
		tmp.SpecialRate, _ = econv.Int32(data["SpecialRate"])
		tmp.BoosRate, _ = econv.Int32(data["BoosRate"])
		tmp.ChestRate, _ = econv.Int32(data["ChestRate"])
		tmp.MonsterPool, _ = econv.Int(data["MonsterPool"])
		tmp.RedpackPool, _ = econv.Int(data["RedpackPool"])
		tmp.SpecialPool, _ = econv.Int(data["SpecialPool"])
		tmp.BossPool, _ = econv.Int(data["BossPool"])
		tmp.ChestPool, _ = econv.Int(data["ChestPool"])

		bulletRateMapConfigs[tmp.Id] = tmp
		//database.AdminDB.Create(tmp)
	}
	GetBulletRateToMap()
	//fmt.Printf("bulletRateConfigs %+v", bulletRateMapConfigs[1])

	bulletMapConfigs = make(map[int]*Bullet)
	datas = excel.LoadExcelMap("conf/excel/Bullet.xlsx")
	for _, data := range datas {
		tmp := &Bullet{}
		tmp.Id, _ = econv.Int(data["Id"])

		tmp.BuffID, _ = econv.Int(data["BuffID"])
		tmp.Probability, _ = econv.Int(data["Probability"])
		tmp.HP, _ = econv.Int(data["HP"])

		bulletMapConfigs[tmp.Id] = tmp
	}
	//for id, c := range bulletRateMapConfigs {
	//	fmt.Printf("bulletRateConfigs %d %+v", id, c)
	//
	//}

}

func GetBulletRateToMap() {
	list := GetAllBulletRate()
	for _, b := range list {
		*bulletRateMapConfigs[b.Id] = b
	}
}
