package data

import (
	"fish_server_2021/game/data/model"
	"lolGF/utils/econv"
	"strings"
	"sync"
)

var allLevelConfigMap map[int]*LevelConfig
var allLevelConfig []*LevelConfig
var allLevelConfigLock sync.RWMutex

type LevelConfig struct {
	ID     int
	Exp    int64
	AllExp int64
	Award  []AwardItem
}

func AllLevelConfigs() []*LevelConfig {
	return allLevelConfig
}
func GetLevelConfig(id int) *LevelConfig {
	allLevelConfigLock.RLock()
	defer allLevelConfigLock.RUnlock()
	return allLevelConfigMap[id]
}
func InitLevelData() {
	list := model.AllLevelConfigs()
	allLevelConfigMap = make(map[int]*LevelConfig)
	allLevelConfig = make([]*LevelConfig, len(list))
	var allExp int64
	for key, item := range list {
		allExp += item.Exp
		tmp := LevelConfig{
			ID:     item.ID,
			Exp:    item.Exp,
			AllExp: allExp,
		}

		awards := strings.Split(item.Award, ";")
		tmp.Award = make([]AwardItem, len(awards))
		for i, str := range awards {
			awardItems := strings.Split(str, ",")

			itemId, _ := econv.Int32(awardItems[0])
			num, _ := econv.Int64(awardItems[1])

			item := AwardItem{
				Id:  itemId,
				Num: num,
			}
			tmp.Award[i] = item
		}
		allLevelConfigMap[tmp.ID] = &tmp
		allLevelConfig[key] = &tmp
	}
}
