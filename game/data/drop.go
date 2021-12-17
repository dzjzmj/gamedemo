package data

import (
	"fish_server_2021/libs/excel"
	"lolGF/utils/econv"
	"sync"
)

var dropGroupConfigs map[int][]*Drop // GroupId => Array
var dropGroupConfigsLock sync.RWMutex

var itemConfigs map[int32]*Item

type Item struct {
	Id   int32 // 物品ID
	Num  int64 // 数量
	Type int32 // 分类
}
type Drop struct {
	Id        int
	GroupId   int
	Item      int32
	Number    int64
	Weight    int
	Broadcast int
}

func GetDropGroupConfigs(groupId int) (config []*Drop, ok bool) {
	dropGroupConfigsLock.RLock()
	config, ok = dropGroupConfigs[groupId]
	dropGroupConfigsLock.RUnlock()
	return
}

func GetItemConfig(id int32) *Item {
	dropGroupConfigsLock.RLock()
	config := itemConfigs[id]
	dropGroupConfigsLock.RUnlock()
	return config
}

func init() {
	dropGroupConfigs = make(map[int][]*Drop)
	itemConfigs = make(map[int32]*Item)
	datas := excel.LoadExcelMap("conf/excel/DropItem.xlsx")
	for _, data := range datas {
		tmp := &Drop{}
		tmp.Id, _ = econv.Int(data["Id"])
		tmp.GroupId, _ = econv.Int(data["GroupId"])
		tmp.Item, _ = econv.Int32(data["Item"])
		tmp.Number, _ = econv.Int64(data["Number"])
		tmp.Weight, _ = econv.Int(data["Weight"])
		tmp.Broadcast, _ = econv.Int(data["Broadcast"])

		_, ok := dropGroupConfigs[tmp.GroupId]
		if !ok {
			dropGroupConfigs[tmp.GroupId] = make([]*Drop, 0, 10)
		}
		dropGroupConfigs[tmp.GroupId] = append(dropGroupConfigs[tmp.GroupId], tmp)

	}
	datas = excel.LoadExcelMap("conf/excel/Item.xlsx")
	for _, data := range datas {
		tmp := &Item{}
		tmp.Id, _ = econv.Int32(data["Id"])
		tmp.Type, _ = econv.Int32(data["Type"])
		itemConfigs[tmp.Id] = tmp
	}
}
