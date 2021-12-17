package internal

import (
	"fish_server_2021/common"
	"fish_server_2021/common/proto"
	"fish_server_2021/game/base"
	"fish_server_2021/game/data"
	"fish_server_2021/game/data/model"
	json "github.com/json-iterator/go"
	"lolGF/log"
	"lolGF/module"
	"lolGF/utils/econv"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const (
	orderMaxForSync = 400 //财富同步

	bulletLiveSec = 5
	skillBulletHP = 10
	// 技能ID
	skillIdPenetrate = 104 // 穿透
	skillIdAim       = 102 // 锁定
	skillIdDouble    = 103 // 分身
	skillIdRage      = 101 // 狂暴
	maxSkillIdRage   = skillIdPenetrate
)

var randSource = rand.New(rand.NewSource(time.Now().UnixNano() + 234560))
var currentBulletHP = 1

var (
	userMap     sync.Map
	userRoomMap sync.Map
	quitRoomMap sync.Map // 保存5分钟内离开的房间
)

type Player struct {
	playerCount       model.PlayerDayCount
	BaseInfo          *common.UserInfo
	Hero              *data.Hero
	nextFire          *data.FirePower
	bulletRates       []*data.BulletRate
	Seat              int
	room              *Room
	currentBulletRate int

	BulletCost  int64 //金币消耗
	BulletAward int64 //打中鱼奖励

	RecordStart    int64
	orderStep      uint16
	orderValue     map[int64]*PlayerBullet //子弹ID--金币值
	orderValueLock sync.RWMutex

	specialDead     map[int64]*PlayerSpecialDead
	specialDeadLock sync.RWMutex

	dropNum         int64
	dropItemNum     map[int32]int64
	dropItemNumLock sync.RWMutex
	lastDropInDay   int
	casino          *Casino
	PlayerRadPack
	quitTimer    *module.Timer
	bulletTimer  *module.Timer
	skillConfig  map[int]*SkillStatus
	doOnceLock   sync.Mutex
	syncOnceLock sync.Mutex
	nextLevel    int
	nextLevelNum int64
}
type PlayerSpecialDead struct {
	monster    *Monster
	rateIndex  int
	createTime int64
	skill      *data.Skill
	num        int
}
type Casino struct {
	Current *data.Casino
	Drops   []*data.Drop
	Data    *model.PlayerCasino
}

func (player *Player) initCasino() {
	if player.casino == nil {

	}
}

type PlayerRadPack struct {
	currAwardIndex int
	waitGetAward   *AwardItem
	upLevelAward   []data.AwardItem
}

type PlayerBullet struct {
	rateIndex  int
	live       int
	times      int
	createTime int64
}

func (player *Player) AddSpecialDead(id int64, monster *Monster, rateIndex int) {
	player.specialDeadLock.Lock()
	defer player.specialDeadLock.Unlock()
	skillId := 0
	if monster.monsterData.Type == data.MonsterTypeBlackHole {
		skillId = 202
	}
	if monster.monsterData.Type == data.MonsterTypeLightning {
		skillId = 201
	}
	player.specialDead[id] = &PlayerSpecialDead{
		monster:    monster,
		rateIndex:  rateIndex,
		createTime: currUnixTime,
	}
	if skillId > 0 {
		skill, ok := data.GetSkillConfig(skillId)
		if ok {
			player.specialDead[id].skill = skill
		}
	}
}
func (player *Player) GetSpecialDead(id int64) *PlayerSpecialDead {
	player.specialDeadLock.Lock()
	defer player.specialDeadLock.Unlock()
	dead, ok := player.specialDead[id]
	if !ok {
		return nil
	}
	t := 5
	if dead.skill != nil {
		t = dead.skill.SkillTime
	}
	if currUnixTime-dead.createTime > int64(t) {
		delete(player.specialDead, id)
		return nil
	}

	//delete(player.specialDead, id)
	return dead
}

func (player *Player) clearAllBullet() {
	isSend := false
	player.orderValueLock.Lock()
	for id, b := range player.orderValue {
		if (currUnixTime-b.createTime) > 1 && b.times == 0 {
			rate := player.bulletRates[b.rateIndex]
			gun := rate.Gun
			player.refundCoin(gun, false)

			isSend = true
			delete(player.orderValue, id)
		}
	}
	player.orderValueLock.Unlock()
	if isSend {
		player.room.notifyAll(SMDropItem, DropItemRet{
			UID:   player.BaseInfo.ID,
			MID:   0,
			ID:    common.CoinItemId,
			Num:   0,
			Total: player.BaseInfo.Coin,
		})
	}
}

// 子弹超时没消耗退费
func (player *Player) bulletTimeout() {
	if player.bulletTimer != nil {
		player.bulletTimer.Stop()
		player.bulletTimer = nil
	}
	player.doOnceLock.Lock()
	defer player.doOnceLock.Unlock()

	isSend := false
	oldCoin := player.BaseInfo.Coin
	if player.BaseInfo.Coin < player.BaseInfo.MaxPower {
		isSend = true
	}
	player.orderValueLock.Lock()
	for id, b := range player.orderValue {
		if (currUnixTime - b.createTime) > bulletLiveSec {
			if b.times == 0 {
				rate := player.bulletRates[b.rateIndex]
				gun := rate.Gun
				player.refundCoin(gun, false)
			}
			delete(player.orderValue, id)
		}
	}
	player.orderValueLock.Unlock()
	if isSend && oldCoin != player.BaseInfo.Coin {
		player.room.notifyAll(SMDropItem, DropItemRet{
			UID:   player.BaseInfo.ID,
			MID:   0,
			ID:    common.CoinItemId,
			Num:   0,
			Total: player.BaseInfo.Coin,
		})
	}

	// 清理特殊怪过期记录
	player.specialDeadLock.Lock()
	defer player.specialDeadLock.Unlock()
	for id, item := range player.specialDead {
		t := 5
		if item.skill != nil {
			t = item.skill.SkillTime
		}
		if currUnixTime-item.createTime > int64(t) {
			delete(player.specialDead, id)
		}
	}
}
func (player *Player) SyncCoin() {
	player.syncOnceLock.Lock()
	defer player.syncOnceLock.Unlock()
	if player.BulletCost == player.BulletAward {
		return
	}

	changeCoin := player.BulletAward - player.BulletCost

	log.Debug("changeCoin %v", changeCoin)
	// 同步财富服接口
	if changeCoin > 0 {
		_ = common.ItemClient.AddItem(proto.ItemReq{
			Uid:     player.BaseInfo.ID,
			ItemId:  common.CoinItemId,
			Num:     changeCoin,
			Action:  1,
			Service: "game",
		})
	} else {
		_ = common.ItemClient.SubItem(proto.ItemReq{
			Uid:     player.BaseInfo.ID,
			ItemId:  common.CoinItemId,
			Num:     -changeCoin,
			Action:  1,
			Service: "game",
		})
	}
	player.BulletCost = 0
	player.BulletAward = 0

	player.RecordStart = currUnixTime
}
func (player *Player) getNewItemNum(id int32) int64 {
	// 返回最新数量
	newNum, _ := common.ItemClient.GetItem(proto.ItemReq{
		Uid:    player.BaseInfo.ID,
		ItemId: id,
	})

	return newNum
}

func (player *Player) addItem(itemId int32, number int64, mid int64, action int, actionData string) {
	var err error
	var newNum int64
	if itemId == common.GemItemId {
		newNum = player.addGem(number, action, actionData, false)
	} else if itemId == common.CoinItemId {
		newNum = player.addCoin(number, false, number)
	} else if itemId == common.PearlItemId {
		newNum = player.addPearl(number, action, actionData, false)
	} else {
		err = common.ItemClient.AddItem(proto.ItemReq{
			Uid:         player.BaseInfo.ID,
			ItemId:      itemId,
			Num:         number,
			Action:      action,
			ActionData1: actionData,
			Service:     "game",
		})
		if err == nil {
			newNum = player.getNewItemNum(itemId)
		}
	}

	if err == nil {
		player.doOnceLock.Lock()
		player.BaseInfo.Items[itemId] = newNum
		player.doOnceLock.Unlock()

		item := data.GetItemConfig(itemId)
		if mid > 0 {
			common.TaskTriggerAddItem(player.BaseInfo.ID, itemId, item.Type, number)
		}
		// 发消息
		player.room.notifyAll(SMDropItem, DropItemRet{
			UID:   player.BaseInfo.ID,
			MID:   mid,
			ID:    itemId,
			Num:   number,
			Total: newNum,
		})

	}
}

func (player *Player) addItemFromOther(itemId int32, number int64, mid int64, action int, actionData string) {
	var newNum int64
	if itemId == common.GemItemId {
		newNum = player.addGem(number, action, actionData, true)
	} else if itemId == common.CoinItemId {
		newNum = player.addCoinFromOther(number)
	} else if itemId == common.PearlItemId {
		newNum = player.addPearl(number, action, actionData, true)
	} else {
		newNum = player.getNewItemNum(itemId)
	}

	player.doOnceLock.Lock()
	player.BaseInfo.Items[itemId] = newNum
	player.doOnceLock.Unlock()
	// 发消息
	if player.room != nil {
		player.room.notifyAll(SMDropItem, DropItemRet{
			UID:   player.BaseInfo.ID,
			MID:   mid,
			ID:    itemId,
			Num:   number,
			Total: newNum,
		})
	}
}

func (player *Player) subItemFromOther(itemId int32, number int64, mid int64, action int, actionData string) {
	var newNum int64
	if itemId == common.GemItemId {
		newNum, _ = player.decGem(number, true)
	} else if itemId == common.CoinItemId {
		newNum, _ = player.decCoinFromOther(number)
	} else if itemId == common.PearlItemId {
		newNum, _ = player.decPearl(number, action, actionData, true)
	} else {
		newNum = player.getNewItemNum(itemId)
	}

	player.doOnceLock.Lock()
	player.BaseInfo.Items[itemId] = newNum
	player.doOnceLock.Unlock()
	// 发消息
	if player.room != nil {
		player.room.notifyAll(SMDropItem, DropItemRet{
			UID:   player.BaseInfo.ID,
			MID:   mid,
			ID:    itemId,
			Num:   -number,
			Total: newNum,
		})
	}
}
func (player *Player) subItem(itemId int32, number int64, mid int64, action int, actionData string) bool {
	var err error
	var newNum int64
	var ok bool
	if itemId == common.GemItemId {
		newNum, ok = player.decGem(number, false)
	} else if itemId == common.CoinItemId {
		newNum, ok = player.decCoin(number)
	} else if itemId == common.PearlItemId {
		newNum, ok = player.decPearl(number, action, actionData, false)
	} else {
		err = common.ItemClient.SubItem(proto.ItemReq{
			Uid:         player.BaseInfo.ID,
			ItemId:      itemId,
			Num:         number,
			Action:      action,
			ActionData1: actionData,
			Service:     "game",
		})
		if err == nil {
			newNum = player.getNewItemNum(itemId)
			ok = true
		} else {
			ok = false
		}
	}

	if ok {
		player.doOnceLock.Lock()
		player.BaseInfo.Items[itemId] = newNum
		player.doOnceLock.Unlock()

		item := data.GetItemConfig(itemId)
		common.TaskTriggerUseItem(player.BaseInfo.ID, itemId, item.Type, number)
		// 发消息
		player.room.notifyAll(SMDropItem, DropItemRet{
			UID:   player.BaseInfo.ID,
			MID:   mid,
			ID:    itemId,
			Num:   -number,
			Total: newNum,
		})

	}
	return ok
}

func (player *Player) addGem(v int64, action int, actionData string, isFromOther bool) int64 {
	newNum := atomic.AddInt64(&player.BaseInfo.Gem, v)
	if !isFromOther {
		skeleton.GoSafe(func() {
			_ = common.ItemClient.AddItem(proto.ItemReq{
				Uid:         player.BaseInfo.ID,
				ItemId:      common.GemItemId,
				Num:         v,
				Action:      action,
				ActionData1: actionData,
				Service:     "game",
			})
		})
	}
	player.upPowerNotice(false)
	return newNum
}
func (player *Player) decGem(v int64, isFromOther bool) (newNum int64, ok bool) {
	if player.BaseInfo.Gem < v {
		return player.BaseInfo.Gem, false
	}
	newNum = atomic.AddInt64(&player.BaseInfo.Gem, -v)
	if !isFromOther {
		skeleton.GoSafe(func() {
			_ = common.ItemClient.SubItem(proto.ItemReq{
				Uid:     player.BaseInfo.ID,
				ItemId:  common.GemItemId,
				Num:     v,
				Action:  1,
				Service: "game",
			})
		})
	}
	return newNum, true
}
func (player *Player) addPearl(v int64, action int, actionData string, isFromOther bool) int64 {
	newNum := atomic.AddInt64(&player.BaseInfo.Pearl, v)
	if !isFromOther {
		skeleton.GoSafe(func() {
			_ = common.ItemClient.AddItem(proto.ItemReq{
				Uid:         player.BaseInfo.ID,
				ItemId:      common.PearlItemId,
				Num:         v,
				Action:      action,
				ActionData1: actionData,
				Service:     "game",
			})
		})
	}
	return newNum
}
func (player *Player) decPearl(v int64, action int, actionData string, isFromOther bool) (newNum int64, ok bool) {
	if player.BaseInfo.Pearl < v {
		return player.BaseInfo.Pearl, false
	}
	newNum = atomic.AddInt64(&player.BaseInfo.Pearl, -v)

	if !isFromOther {
		skeleton.GoSafe(func() {
			_ = common.ItemClient.SubItem(proto.ItemReq{
				Uid:         player.BaseInfo.ID,
				ItemId:      common.PearlItemId,
				Num:         v,
				Action:      action,
				ActionData1: actionData,
				Service:     "game",
			})
		})
	}
	return newNum, true
}

//扣除金币
func (player *Player) decCoin(dec int64) (newNum int64, ok bool) {
	if player.BaseInfo.Coin < dec {
		return player.BaseInfo.Coin, false
	}

	newNum = atomic.AddInt64(&player.BaseInfo.Coin, -dec)
	player.onCoinChange()
	return newNum, true
}
func (player *Player) decCoinFromOther(dec int64) (newNum int64, ok bool) {
	if player.BaseInfo.Coin < dec {
		return player.BaseInfo.Coin, false
	}

	newNum = atomic.AddInt64(&player.BaseInfo.Coin, -dec)
	return newNum, true
}

func (player *Player) recharge(add int64) {
	atomic.AddInt64(&player.BaseInfo.Coin, add)
	atomic.AddInt64(&player.BulletAward, add)
}

// 添加金币(打中鱼所得)
func (player *Player) addCoin(add int64, sync bool, original int64) int64 {
	if add <= 0 {
		return player.BaseInfo.Coin
	}
	newNum := atomic.AddInt64(&player.BaseInfo.Coin, add)
	atomic.AddInt64(&player.BulletAward, add)
	atomic.AddInt64(&player.playerCount.CoinAddSum, add)
	atomic.AddInt64(&player.playerCount.CoinAddOriginal, original)
	//player.BaseInfo.Coin += add
	//player.BulletAward += add
	//player.coinAddSum += add

	if sync {
		player.onCoinChange()
	}
	return newNum
}
func (player *Player) addCoinFromOther(add int64) int64 {
	if add <= 0 {
		return player.BaseInfo.Coin
	}
	newNum := atomic.AddInt64(&player.BaseInfo.Coin, add)
	return newNum
}
func (player *Player) refundCoin(add int64, sync bool) bool {
	if add <= 0 {
		return false
	}
	atomic.AddInt64(&player.BaseInfo.Coin, add)
	atomic.AddInt64(&player.BulletAward, add)
	atomic.AddInt64(&player.playerCount.CoinDecSum, -add)
	//atomic.AddInt32(&player.coinAddSum, add)
	//player.BaseInfo.Coin += add
	//player.BulletAward += add
	//player.coinAddSum += add

	if sync {
		player.onCoinChange()
	}

	return true
}
func (player *Player) onCoinChange() {
	syncVCMark := player.orderStep >= orderMaxForSync
	if !syncVCMark && currUnixTime-player.RecordStart > 80 {
		syncVCMark = true
	}
	flag := false
	if syncVCMark {
		player.orderStep = 0
		player.bulletTimeout()
		flag = true
		player.SyncCoin()
	} else {
		r := randSource.Intn(100)
		if r <= 15 {
			player.bulletTimeout()
			flag = true
		}
	}
	if !flag {
		if player.BaseInfo.Coin < player.BaseInfo.MaxPower {
			player.bulletTimeout()
		}
	}
}

//添加订单
func (player *Player) AddOrder(order int64) bool {

StartLabel:
	//目前扣除的数量是炮等级
	if player.currentBulletRate > len(player.bulletRates)-1 {
		return false
	}
	rate := player.bulletRates[player.currentBulletRate]
	gun := rate.Gun
	if _, ok := player.decCoin(gun); !ok {
		if player.BaseInfo.Coin < player.BaseInfo.MaxPower {
			player.clearAllBullet()
		}
		if player.currentBulletRate > 0 && player.BaseInfo.Coin > 0 {
			// 降低火力
			//player.changePower(&ChangePowerReq{CT: ChangePowerDec})
			player.decPowerToUsable()
			goto StartLabel
		}
		return false
	}
	//player.BulletCost += gun
	//player.coinDecSum += gun
	atomic.AddInt64(&player.BulletCost, gun)
	atomic.AddInt64(&player.playerCount.CoinDecSum, gun)

	times := 10
	if player.Hero != nil {
		times = player.Hero.BulletHP + player.Hero.Bullet.HP
		if currentBulletHP > times {
			times = currentBulletHP
		}
	}

	pb := PlayerBullet{
		rateIndex:  player.currentBulletRate,
		live:       times,
		createTime: currUnixTime,
		times:      0,
	}
	player.orderValueLock.Lock()
	player.orderValue[order] = &pb
	player.orderValueLock.Unlock()
	//if one.Record != nil {
	//	one.Record.ShootMap[gunLev]++
	//}
	player.orderStep++

	if player.BaseInfo.Coin < player.BaseInfo.MaxPower {
		if player.bulletTimer == nil {
			player.bulletTimer = skeleton.AfterFunc((bulletLiveSec+1)*time.Second, func() {
				player.bulletTimeout()
			})
		}
	}

	return true
}

//移除订单
func (player *Player) DecOrderValue(order int64) int {
	player.orderValueLock.RLock()
	tmp, ok := player.orderValue[order]
	player.orderValueLock.RUnlock()

	if !ok {
		return -1
	}
	if tmp.live <= 0 {
		player.deleteOrderValue(order)
		base.SendMsgToGate(player.BaseInfo.ID, SMReadPackBulletNum, RedPackBulletNum{BulletNum: player.BaseInfo.BulletNum})
		return -1
	}
	tmp.live--
	if tmp.times > 1 {
		// 第二次开始需要扣钱
		rate := player.bulletRates[tmp.rateIndex]
		gun := rate.Gun
		if _, ok := player.decCoin(gun); !ok {
			player.deleteOrderValue(order)
			base.SendMsgToGate(player.BaseInfo.ID, SMReadPackBulletNum,
				RedPackBulletNum{BulletNum: player.BaseInfo.BulletNum})
			return -1
		}
	}
	tmp.times++
	if tmp.live <= 0 {
		player.deleteOrderValue(order)
	}
	return tmp.rateIndex
}
func (player *Player) deleteOrderValue(order int64) {
	player.orderValueLock.Lock()
	defer player.orderValueLock.Unlock()
	delete(player.orderValue, order)
}
func (player *Player) getPool() {
	base.SendMsgToGate(player.BaseInfo.ID, common.CMPlayerPool, map[string]interface{}{
		"个人水池":     player.playerCount.Pool,
		"登录天数":     player.BaseInfo.LoginDay,
		"当天获得金币税后": player.playerCount.CoinAddSum,
		"当天获得金币":   player.playerCount.CoinAddOriginal,
		"当天消耗金币":   player.playerCount.CoinDecSum,
	})
}
func getCacheUser(uid int32) *Player {
	value, ok := userMap.Load(uid)
	if !ok || value == nil {
		return nil
	}

	one := value.(*Player)
	return one
}

func delCacheUser(uid int32) {
	userMap.Delete(uid)
}

func getUser(uid int32, getNewInfo bool) *Player {
	one := getCacheUser(uid)
	if one != nil {
		isSame := time.Unix(one.playerCount.UpdateTime, 0).Day() == currTimeInDay
		if !isSame {
			//log.Debug("isSame %v", isSame)
			one = nil
		}
	}
	isUpdate := false
	nowDate := time.Now().Format("2006-01-02")
	isInit := false
	if one == nil {
		isUpdate = true
		ui := common.GetUserInfo(uid)
		if ui == nil {
			return nil
		}

		one = &Player{
			BaseInfo:    ui,
			Seat:        -1,
			orderValue:  make(map[int64]*PlayerBullet, 200),
			skillConfig: make(map[int]*SkillStatus),
			specialDead: make(map[int64]*PlayerSpecialDead),
			dropItemNum: make(map[int32]int64),
		}

		one.skillConfig[101] = &SkillStatus{}
		one.skillConfig[102] = &SkillStatus{}
		one.skillConfig[103] = &SkillStatus{}
		one.skillConfig[104] = &SkillStatus{}

		playerCount := model.GetPlayerDayCount(nowDate, uid)
		one.playerCount = playerCount
		_ = json.UnmarshalFromString(playerCount.DropItem, &one.dropItemNum)
		if playerCount.ID == 0 {
			isInit = true
		}
		userMap.Store(uid, one)
	} else {
		if getNewInfo && one.room == nil {
			isUpdate = true
			ui := common.GetUserInfo(uid)
			if ui == nil {
				return nil
			}
			one.BaseInfo = ui
		}
	}
	if isUpdate {
		hero, ok := data.GetHeroConfig(one.BaseInfo.Hero)
		if ok {
			one.Hero = hero
		}
		// 下一级火力
		fires := data.GetFirePowerConfigs()
		for _, fire := range fires {
			if fire.Power > one.BaseInfo.MaxPower {
				one.nextFire = fire
				break
			}
		}
	}
	//log.Debug("getPlayer %v", isInit)
	if one.playerCount.UpdateTime > 0 || isInit {
		if time.Unix(one.playerCount.UpdateTime, 0).Format("2006-01-02") != nowDate || isInit {
			//log.Debug("getPlayer2 %v", isInit)
			var playerCount model.PlayerDayCount
			playerCount.Date = nowDate
			playerCount.UID = one.BaseInfo.ID
			playerCount.Day = one.BaseInfo.LoginDay
			playerCount.CreateAt = time.Now()
			// 初始玩家水池
			pools, ok := data.GetNewPlayerPoolByDay(one.BaseInfo.LoginDay)
			if !ok {
				pools, _ = data.GetNewPlayerPoolByDay(data.NewPlayerMaxDay)
			}
			for _, pool := range pools {
				if pool.UpperLimit > playerCount.Pool {
					playerCount.Pool = pool.UpperLimit
				}
			}
			_ = playerCount.Create()
			one.playerCount = playerCount

			if one.BaseInfo.LoginDay > 1 {
				ld, _ := econv.String(one.BaseInfo.LoginDay)
				_ = common.ItemClient.AddItem(proto.ItemReq{
					Uid:         uid,
					ItemId:      common.CoinItemId,
					Num:         common.InitItems[common.CoinItemId],
					Action:      5,
					ActionData1: ld,
					Service:     "game",
				})
				one.BaseInfo.Coin += common.InitItems[common.CoinItemId]
			}
		}
	}
	//log.Debug("getPlayer %+v", one.BaseInfo)
	return one
}
