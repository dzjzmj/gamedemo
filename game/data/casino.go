package data

import (
	"context"
	"fish_server_2021/game/data/model"
	"fish_server_2021/libs/excel"
	"fish_server_2021/libs/redis"
	"lolGF/utils/econv"
)

type Casino struct {
	ID           int
	Name         string
	Level        int
	Icon         string
	Backboard    string
	AwardNum     int
	AwardGroupId int
	NeedScore    int
	LevelAward   []AwardItem
}

var casinos []*Casino

func init() {
	casinos = make([]*Casino, 0, 100)

	datas := excel.LoadExcelMap("conf/excel/Casino.xlsx")
	for _, data := range datas {
		tmp := Casino{}
		tmp.ID, _ = econv.Int(data["Id"])
		tmp.Name = data["Name"]
		tmp.Level, _ = econv.Int(data["Level"])
		tmp.Icon = data["Icon"]
		tmp.Backboard = data["Backboard"]
		tmp.AwardNum, _ = econv.Int(data["AwardNum"])
		tmp.AwardGroupId, _ = econv.Int(data["AwardGroupId"])
		tmp.NeedScore, _ = econv.Int(data["NeedScore"])
		tmp.LevelAward = ItemString2Struct(data["LevelAward"])

		casinos = append(casinos, &tmp)
	}
}

func GetCasinos() []*Casino {
	return casinos
}

func GetPlayerCasino(id int32) *model.PlayerCasino {
	dbData := model.GetPlayerCasino(id)
	return dbData
}

func SavePlayerCasino(pc *model.PlayerCasino) {
	uidStr, _ := econv.String(pc.ID)
	redis.RClient.HMSet(context.Background(), "PlayerCasino"+uidStr, "c", pc.CurrentLevel, "g", pc.GotIds)
}

func SetGot(pc *model.PlayerCasino, key uint32) {
	mark := uint32(1) << key
	pc.GotIds |= mark
}

func GetGot(pc *model.PlayerCasino, list []*Drop) map[int]bool {
	retMap := make(map[int]bool)
	for key := range list {
		mark := uint32(1) << key
		retMap[key] = pc.GotIds&mark != 0
	}
	return retMap
}
