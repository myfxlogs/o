// facesnap_whitelist_ops.go —— 面基线册 ops 键域补册（工单#26444）：
// 主册 facesnap_whitelist.go 贴 check-lines 300 行棘轮线——deploy/
// okxgrid 键域拆件先例同形，init 并入同包 map。
package hunger

func init() {
	// 工单#26444：ops 键域=fin_0222 种 ops.unit_registry（建制单元
	// 注册表）在册即面，读者册载建制单元查件/差分探针（unitdrift.go
	// cfg.Get 实读+unit_watch.sh psql 实读同域）。
	faceBaselineWhitelist["face:dep_pair:config:ops→建制单元查件/差分探针"] = "工单#26444 入册——ops.unit_registry=fin_0222 种建制单元注册表在册即面，读者=建制单元查件/差分探针（internal/portfolio/hunger/unitdrift.go cfg.Get 实读+scripts/unit_watch.sh psql 实读同键）；枚举对 depConfigReaders 册内合法新面登记不重复升"
	faceBaselineWhitelist["face:gate_domain:ops"] = "工单#26444 入册——ops app_config 键域=建制单元注册表 owner 类在册即面（写域册在案，读者=建制单元查件/差分探针实读在案）；gate_domain 枚举=split_part 本体零硬编码合法新面登记不重复升"
}
