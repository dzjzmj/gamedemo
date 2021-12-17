package data

import (
	"fish_server_2021/libs/database"
	"fish_server_2021/libs/excel"
	"lolGF/utils/econv"
	"sync"
)

var poolsConfig map[int]map[int]*Pools // Room => Map[Id => Pools]
var poolsMapConfig map[int]Pools
var poolsConfigLock sync.RWMutex

var newPlayerPoolConfigs []*NewPlayerPool
var newPlayerPoolByDay map[int][]*NewPlayerPool
var NewPlayerMaxDay = 1

func GetNewPlayerPoolConfigs() []*NewPlayerPool {
	return newPlayerPoolConfigs
}

func GetNewPlayerPoolByDay(day int) (pool []*NewPlayerPool, ok bool) {
	poolsConfigLock.RLock()
	pool, ok = newPlayerPoolByDay[day]
	poolsConfigLock.RUnlock()
	return
}

type Pools struct {
	ID        int
	Pool      int64
	PoolItems []PoolItem
}
type PoolItem struct {
	ID          int
	PoolId      int
	LowerLimit  int64
	UpperLimit  int64
	Coefficient int32
}

func initPool() {
	poolsMapConfig = make(map[int]Pools)

	// from excel
	//datas := excel.LoadExcelMap("conf/excel/PoolInfluence.xlsx")
	//for _, data := range datas {
	//	tmp := PoolItem{}
	//	tmp.PoolId, _ = econv.Int(data["PoolId"])
	//	tmp.LowerLimit, _ = econv.Int64(data["LowerLimit"])
	//	tmp.UpperLimit, _ = econv.Int64(data["UpperLimit"])
	//	tmp.Coefficient, _ = econv.Int32(data["Coefficient"])
	//	pool, ok := poolsMapConfig[tmp.PoolId]
	//	if !ok {
	//		pool = Pools{
	//			ID:        tmp.PoolId,
	//			PoolItems: make([]PoolItem, 0, 10),
	//		}
	//	}
	//	pool.PoolItems = append(pool.PoolItems, tmp)
	//	poolsMapConfig[tmp.PoolId] = pool
	//}
	// from db
	var poolItems []PoolItem
	database.AdminDB.Order("id asc").Find(&poolItems)
	for _, tmp := range poolItems {
		pool, ok := poolsMapConfig[tmp.PoolId]
		if !ok {
			pool = Pools{
				ID:        tmp.PoolId,
				PoolItems: make([]PoolItem, 0, 10),
			}
		}
		pool.PoolItems = append(pool.PoolItems, tmp)
		poolsMapConfig[tmp.PoolId] = pool
	}

	newPlayerPoolByDay = make(map[int][]*NewPlayerPool)
	newPlayerPoolConfigs = make([]*NewPlayerPool, 0, 20)
	datas := excel.LoadExcelMap("conf/excel/NoviciatePools.xlsx")
	for _, data := range datas {
		tmp := &NewPlayerPool{}
		tmp.ID, _ = econv.Int(data["Id"])
		tmp.Day, _ = econv.Int(data["Day"])
		tmp.LowerLimit, _ = econv.Int64(data["LowerLimit"])
		tmp.UpperLimit, _ = econv.Int64(data["UpperLimit"])
		tmp.Coefficient, _ = econv.Int32(data["Coefficient"])

		newPlayerPoolConfigs = append(newPlayerPoolConfigs, tmp)

		_, ok := newPlayerPoolByDay[tmp.Day]
		if !ok {
			newPlayerPoolByDay[tmp.Day] = make([]*NewPlayerPool, 0, 10)
		}
		newPlayerPoolByDay[tmp.Day] = append(newPlayerPoolByDay[tmp.Day], tmp)

		if tmp.Day > NewPlayerMaxDay {
			NewPlayerMaxDay = tmp.Day
		}
	}
}

type NewPlayerPool struct {
	ID          int
	Day         int
	LowerLimit  int64
	UpperLimit  int64
	Coefficient int32
}

func GetPools(roomType int, id int) (pool *Pools, ok bool) {
	poolsConfigLock.RLock()
	pool, ok = poolsConfig[roomType][id]
	poolsConfigLock.RUnlock()
	return
}

func GetAllPools(roomType int) (pool map[int]*Pools, ok bool) {
	poolsConfigLock.RLock()
	pool, ok = poolsConfig[roomType]
	poolsConfigLock.RUnlock()
	return
}

func SetAllPools(roomType int, pool map[int]*Pools) {
	poolsConfigLock.Lock()
	poolsConfig[roomType] = pool
	poolsConfigLock.Unlock()
	return
}
