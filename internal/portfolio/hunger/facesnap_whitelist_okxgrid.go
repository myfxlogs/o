// facesnap_whitelist_okxgrid.go —— 面基线册 okxgrid 键域补册（工单#25840）：
// 主册 facesnap_whitelist.go 贴 check-lines 300 行棘轮线——deploy 键域拆件
// 同例（facesnap_whitelist_deploy.go），init 并入同包 map。
package hunger

func init() {
	// 工单#25840：okxgrid 键域=fin_0219 种五键（snap_sec/max_invest_usd/
	// kill_stop_type/harvest_min_usd/live_switch）在册即面，读者册载
	// 网格bot执行环（okxgrid.Runner cfg.GetFloat 实读同域）。
	faceBaselineWhitelist["face:dep_pair:config:okxgrid→网格bot执行环"] = "工单#25840 入册——okxgrid 键域=fin_0219 种五键（snap_sec/max_invest_usd/kill_stop_type/harvest_min_usd/live_switch）在册即面，读者册载网格bot执行环（internal/trading/okxgrid/run.go cfg.GetFloat 实读 okxgrid.snap_sec 同域）；枚举对 depConfigReaders 册内合法新面登记不重复升"
	faceBaselineWhitelist["face:gate_domain:okxgrid"] = "工单#25840 入册——okxgrid app_config 键域=venue原生网格bot参数+业主首闸在册即面（写域册 dm/owner 双类在案，读者=网格bot执行环实读在案）；gate_domain 枚举=split_part 本体零硬编码合法新面登记不重复升"
}
