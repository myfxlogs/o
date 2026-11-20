// facesnap_whitelist_deploy.go —— 面基线册 deploy 键域补册（工单#25760）：
// 主册 facesnap_whitelist.go 贴 check-lines 300 行棘轮线（续增即撞），
// 同 facesnap.go→facesnap_whitelist.go 拆件先例——新面册条以 init 并入
// 同包 map（顶层 var 初始化先于 init，读面 faceBaselineExempt 不变）。
package hunger

func init() {
	// deploy 键域=app_config deploy.ocore_window_utc/max_hold_h（fin_0217
	// 种键，写域册 dm 类；读者=scripts/deploy_watch.sh 部署管窗 psql 实读
	// 两键）。不登记则 ④族枚举对落 config:deploy→∅ orphan 形误升件
	// （depblind 桩测实证：TestProbeDepsEmptyEnumFailClosed 等六案红）。
	faceBaselineWhitelist["face:dep_pair:config:deploy→部署管窗"] = "2026-11-20 入册——deploy 键域=app_config deploy.ocore_window_utc（o-core 换件 UTC 窗）+deploy.ocore_window_max_hold_h（越窗阈）在册即面，读者册载部署管窗（scripts/deploy_watch.sh psql 实读两键，窗内放件/窗外入窗队列/越窗升件径在案）；depConfigReaders 补册后枚举对合法新面登记不重复升；工单#25760"
	// gate_domain 面=键域谱面本体零硬编码（split_part 枚举），域在册恒
	// 产非产消断——同 repair/okxws 域先例登记不重复升。
	faceBaselineWhitelist["face:gate_domain:deploy"] = "2026-11-20 入册——deploy app_config 键域=o-core 换件部署窗两键在册即面（写域册 dm 类系统自裁部署时窗，读者=部署管窗实读在案）；gate_domain 枚举=split_part 本体零硬编码合法新面登记不重复升；工单#25760"
}
