package main

// nav_gate.go —— 工单#24802 NAV 完备性闸（行棘轮拆件——service.go 超
// 300 基线禁增量）。缺腿闸单函数收编，nav() 闭包闸点一行接件。

import (
	"fmt"
	"strings"

	"o/internal/gen/dashboardv1"
)

// navBy 完备性闸+按币聚合一体件：o.Money note 行在=腿读败缺腿（部署窗
// funding REST 挂→USDT=1014 坏帧落 md_equity_curve→NAV 假崩实证）——
// 缺腿→返 err 走 report navSnapshot 现成读败让位径本拍零写，err 载缺
// 腿区名可溯；全腿齐→按币聚合账+NAV 读面行。
func navBy(o *dashboardv1.OverviewResponse) (map[string]float64, error) {
	var miss []string
	for _, m := range o.Money {
		if m.GetNote() != "" {
			miss = append(miss, m.GetSection()+"/"+m.GetLabel())
		}
	}
	if len(miss) > 0 {
		return nil, fmt.Errorf("nav 缺腿（%s）——本拍让位不写", strings.Join(miss, ","))
	}
	by := map[string]float64{}
	for _, m := range o.Money {
		if m.GetCcy() != "" && m.GetNote() == "" {
			by[m.GetCcy()] += m.GetAmount()
		}
	}
	if o.NavTotal > 0 {
		by["NAV"] = o.NavTotal // USD 聚合读面续供——按币簿记行另载（升件#10438）
	}
	return by, nil
}
