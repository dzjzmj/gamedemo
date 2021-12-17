package internal

import (
	"math/rand"
	"sort"
)

func weightRandom(weights []int) int {
	rand.Seed(currUnixTime)
	if len(weights) == 1 {
		return 0
	}
	var sum = 0
	for _, w := range weights {
		sum += w
	}
	r := rand.Intn(sum)
	var t = 0
	for i, w := range weights {
		t += w
		if t > r {
			return i
		}
	}
	return len(weights) - 1
}
func weightRandomIndex(weights map[int]int) int {
	saveIndex := make(map[int][]int)
	weightSlice := make([]int, 0, 10)
	for k, v := range weights {
		weightSlice = append(weightSlice, v)
		_, ok := saveIndex[v]
		if !ok {
			saveIndex[v] = make([]int, 0, 5)
		}
		saveIndex[v] = append(saveIndex[v], k)
	}

	sort.Sort(sort.IntSlice(weightSlice))

	i := weightRandom(weightSlice)
	ks := saveIndex[weightSlice[i]]
	//log.Debug("%+v %v\n", weights, ks)
	if len(ks) == 1 {
		return ks[0]
	} else {
		return ks[rand.Intn(len(ks))]
	}
}

func slice2map(s []int) map[int]int {
	r := make(map[int]int)
	for i, v := range s {
		r[i] = v
	}
	return r
}

func inArray(v int, array []int) bool {
	for _, i := range array {
		if i == v {
			return true
		}
	}
	return false
}
