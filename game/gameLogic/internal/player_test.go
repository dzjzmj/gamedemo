package internal

import (
	"fish_server_2021/common"
	"fmt"
	"lolGF/conn"
	"testing"
)

func TestRpcGetUserInfo(t *testing.T) {
	conn.ConnectNats("test")

	u := common.GetUserInfo(1)
	fmt.Println(u)
}

func TestWeightRand(t *testing.T) {
	//r := []int{50, 50}
	//fmt.Println(weightRandomIndex(r))
}

func TestNewId(t *testing.T) {
	weight := 20000
	rateByMonster := 0
	poolCoefficient := 0
	playerWeight := 10000
	prob := weight * (int(totalWeight) + rateByMonster + poolCoefficient + playerWeight) / int(totalWeight)

	fmt.Println(prob)
}
