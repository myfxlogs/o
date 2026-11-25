// depenum_margin.go —— depConfigReaders margin 域补册（工单O-40707）：
// 主册 depenum.go 贴 check-lines 300 行棘轮线——init 并入同包 map
// （depenum_okxgrid/depenum_annualized 拆件先例同形）。
package hunger

func init() {
	// margin.* 五键（collect_ivl_s/exec_enabled/ratio_warn/ratio_kill/
	// max_open_usd——fin_0264 种键）读者=cmd/core marginBeat/marginShortOpen
	// cfg.GetFloat/GetBool 实读同域；册外登记防 config:margin→∅ orphan 误报。
	depConfigReaders["margin"] = "margin-short 采集/执行/健康闸消费面"
}
