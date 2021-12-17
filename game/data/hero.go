package data

import (
	"fish_server_2021/libs/excel"
	"lolGF/utils/econv"
	"sync"
)

var heroConfigs map[int32]*Hero
var heroConfigsLock sync.RWMutex

type Hero struct {
	Id             int32
	Bullet         *Bullet
	RageBullet     *Bullet
	FuryTime       int
	AimTime        int
	BilocationTime int
	PenetrateTime  int
	BulletHP       int
}

func GetHeroConfig(id int32) (configs *Hero, ok bool) {
	heroConfigsLock.RLock()
	configs, ok = heroConfigs[id]
	heroConfigsLock.RUnlock()
	return
}

func initHero() {
	heroConfigs = make(map[int32]*Hero)
	dataFromExcel := excel.LoadExcelMap("conf/excel/Hero.xlsx")
	for _, data := range dataFromExcel {
		tmp := &Hero{}
		tmp.Id, _ = econv.Int32(data["Id"])
		bid, _ := econv.Int(data["Bullet"])
		if bid > 0 {
			b, ok := GetBulletConfigs(bid)
			if ok {
				tmp.Bullet = b
			}
		}
		bid, _ = econv.Int(data["RageBullet"])
		if bid > 0 {
			b, ok := GetBulletConfigs(bid)
			if ok {
				tmp.RageBullet = b
			}
		}

		tmp.FuryTime, _ = econv.Int(data["FuryTime"])
		tmp.AimTime, _ = econv.Int(data["AimTime"])
		tmp.BilocationTime, _ = econv.Int(data["BilocationTime"])
		tmp.PenetrateTime, _ = econv.Int(data["PenetrateTime"])
		tmp.BulletHP, _ = econv.Int(data["BulletHP"])

		heroConfigs[tmp.Id] = tmp
	}
}
