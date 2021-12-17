package data

import (
	"fish_server_2021/libs/excel"
	"fmt"
	"testing"
)

func TestExcel(t *testing.T) {
	datas := excel.LoadExcelMap("conf/excel/BulletRate.xlsx")
	for i, row := range datas {
		fmt.Printf("%d %+v\n", i, row)

	}
}
