package data

import (
	"fish_server_2021/libs/excel"
	"lolGF/utils/econv"
	"strings"
	"sync"
)

var buffConfig map[int]*Buff
var buffConfigsLock sync.RWMutex
var skillConfig map[int]*Skill

type Buff struct {
	Id       int
	Name     string
	Type     int
	Object   int
	Duration int64
	TypeData int64
}

type Skill struct {
	Id           int
	Type         int
	Object       int
	Scope        int
	Influence    int
	SkillTime    int
	CD           int
	ConsumeItems []Item
	BuffId       int
	Probability  int
	EffectsId    int
}

func GetBuffConfig(id int) (configs *Buff, ok bool) {
	buffConfigsLock.RLock()
	configs, ok = buffConfig[id]
	buffConfigsLock.RUnlock()
	return
}

func GetSkillConfig(id int) (configs *Skill, ok bool) {
	buffConfigsLock.RLock()
	configs, ok = skillConfig[id]
	buffConfigsLock.RUnlock()
	return
}

func GetSkill(id int, heroId int32) *Skill {
	s, ok := GetSkillConfig(id)
	if ok {
		skill := *s

		hero, ok := GetHeroConfig(heroId)
		if ok {
			var t int
			if id == 101 {
				t = hero.FuryTime
			} else if id == 102 {
				t = hero.AimTime
			} else if id == 103 {
				t = hero.BilocationTime
			} else if id == 104 {
				t = hero.PenetrateTime
			}
			skill.SkillTime += t
			skill.CD += t

			return &skill
		}
	}
	return nil
}

func init() {
	buffConfig = make(map[int]*Buff)
	dataFromExcel := excel.LoadExcelMap("conf/excel/Buff.xlsx")
	for _, data := range dataFromExcel {
		tmp := &Buff{}
		tmp.Id, _ = econv.Int(data["Id"])
		tmp.Type, _ = econv.Int(data["Type"])
		tmp.Object, _ = econv.Int(data["Object"])
		tmp.Duration, _ = econv.Int64(data["Duration"])
		tmp.TypeData, _ = econv.Int64(data["TypeData"])

		buffConfig[tmp.Id] = tmp
	}
	skillConfig = make(map[int]*Skill)
	dataFromExcel = excel.LoadExcelMap("conf/excel/Skill.xlsx")
	for _, data := range dataFromExcel {
		tmp := &Skill{}
		tmp.Id, _ = econv.Int(data["Id"])
		tmp.Type, _ = econv.Int(data["Type"])
		tmp.Object, _ = econv.Int(data["Object"])
		tmp.Scope, _ = econv.Int(data["Scope"])
		tmp.Influence, _ = econv.Int(data["Influence"])
		tmp.SkillTime, _ = econv.Int(data["SkillTime"])
		tmp.CD, _ = econv.Int(data["CD"])

		if data["ConsumeItem"] != "" {
			consumes := strings.Split(data["ConsumeItem"], ";")
			tmp.ConsumeItems = make([]Item, len(consumes))

			for i, str := range consumes {
				iItems := strings.Split(str, ",")

				itemId, _ := econv.Int32(iItems[0])
				num, _ := econv.Int64(iItems[1])

				item := Item{
					Id:  itemId,
					Num: num,
				}
				tmp.ConsumeItems[i] = item
			}
		}
		tmp.BuffId, _ = econv.Int(data["BuffId"])
		tmp.Probability, _ = econv.Int(data["Probability"])
		tmp.EffectsId, _ = econv.Int(data["EffectsId"])

		skillConfig[tmp.Id] = tmp
	}
}
