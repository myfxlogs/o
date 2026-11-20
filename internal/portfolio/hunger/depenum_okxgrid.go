// depenum_okxgrid.go —— depConfigReaders okxgrid 域补册（工单#25840）：
// 主册 depenum.go 贴 check-lines 300 行棘轮线——init 并入同包 map。
package hunger

func init() {
	// T-0526 工单#25840：okxgrid 键域（snap_sec/max_invest_usd/kill_stop_type/
	// harvest_min_usd/live_switch 五种在册）读者=网格bot执行环（okxgrid.Runner
	// cfg.GetFloat 实读同域）。
	depConfigReaders["okxgrid"] = "网格bot执行环"
}
