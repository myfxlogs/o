// depenum_mt5venue.go —— depConfigReaders CFD 户域补册（工单O-37448）：
// 主册 depenum.go 贴 check-lines 300 行棘轮线（depenum_ops.go 拆件先例
// 同形）——新域条以 init 并入同包 map。
package hunger

func init() {
	// xmglobal.session_token 读者=internal/mt5 sharedSessionRead（汇市巡
	// owner 按户建/<venue>.session_token 共享行——O-37448 venue 寻址会话
	// 面；消费腿=OCI 桥腿 bridge.<venue>.symbols 编目+OpenSession/mt5LazyCli
	// venue 参数化复挂）；xmglobal.mt5.symbols 读者=internal/mt5
	// symbols.go 三源序①显式册（cfg.Get 实读）。
	// 他 proto='mt5' 户（icmarkets 等）首现 app_config 键域=orphan 初报
	// 一次→人工补册同例（域名=roster venue token 动态枚举不可静态预列
	// ——册内只钉 roster 种子在册户）。
	depConfigReaders["xmglobal"] = "CFD 户会话/订阅面"
	// exness 同域补册（O-40205 实证——fin_0257 venue 改名 mt5→exness 后
	// 产者写 exness.* 键域，册无 exness 条=dep_blind config:exness→∅）。
	depConfigReaders["exness"] = "CFD 户会话/订阅面"
}
