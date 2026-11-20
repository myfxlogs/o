// depenum_ops.go —— depConfigReaders ops 域补册（工单#26444）：
// 主册 depenum.go 贴 check-lines 300 行棘轮线（depenum_okxgrid.go
// 拆件先例同形）——新域条以 init 并入同包 map。
package hunger

func init() {
	// 工单#26444：ops.unit_registry 读者=internal/portfolio/hunger/
	// unitdrift.go 建制单元差分探针（cfg.Get 实读）+scripts/
	// unit_watch.sh 本机查件（psql 实读同键）——两读面同域在册。
	depConfigReaders["ops"] = "建制单元查件/差分探针"
}
