package data

import (
	"fish_server_2021/game/data/model"
	"lolGF/utils/econv"
	"strings"
)

var redPackAwardConfigs []*RedPackAward

type RedPackAward struct {
	Id          int
	Name        string
	Class       int
	AddNumber   int64
	AccNumber   int64 // 累计需要数量
	AwardItem   []AwardItem
	Probability []int
}
type AwardItem struct {
	Id  int32 // 物品ID
	Num int64 // 数量
}

func GetRedPackAwardConfigs() []*RedPackAward {
	return redPackAwardConfigs
}

func InitRadPack() {
	redPackAwardConfigs = make([]*RedPackAward, 0, 12)
	//data := excel.LoadExcelMap("conf/excel/RedPacket.xlsx")
	var accNumber int64
	list := model.GetRedPacks()
	for _, data := range list {
		tmp := &RedPackAward{}
		tmp.Id = data.Id
		tmp.Name = data.Name
		tmp.AddNumber = data.AddNumber
		tmp.AccNumber = accNumber + tmp.AddNumber
		accNumber = tmp.AccNumber
		tmp.Class = data.Class

		tmp.AwardItem = ItemString2Struct(data.AwardItem)

		probabilities := strings.Split(data.Probability, ";")
		tmp.Probability = make([]int, len(probabilities))
		for i, str := range probabilities {
			tmp.Probability[i], _ = econv.Int(str)
		}

		redPackAwardConfigs = append(redPackAwardConfigs, tmp)
	}
}

func ItemString2Struct(s string) []AwardItem {
	awards := strings.Split(s, ";")
	awardItems := make([]AwardItem, 0, len(awards))
	for _, str := range awards {
		awardItem := strings.Split(str, ",")
		if len(awardItem) != 2 {
			continue
		}

		itemId, _ := econv.Int32(awardItem[0])
		num, _ := econv.Int64(awardItem[1])

		item := AwardItem{
			Id:  itemId,
			Num: num,
		}
		awardItems = append(awardItems, item)
	}
	return awardItems
}
