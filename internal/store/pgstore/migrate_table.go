package pgstore

// 迁移登记表——版本号唯一（门禁 check_migration_contract.sh ④b 查重锚）。
var migrationTable = []struct {
	Version int
	Name    string
}{
	{1, "0001_app_config.sql"},
	{2, "0002_fin_employee_registry.sql"},
	{3, "0003_market_facts.sql"},
	{4, "0004_seed_okx_symbols.sql"},
	{5, "0005_private_facts.sql"},
	{6, "0006_research.sql"},
	{7, "0007_lifecycle.sql"},
	{8, "0008_risk.sql"},
	// 9 留给 T-0014/0009_execution.sql（在途——版本主键防撞号，合并序 T-0014→T-0015）。
	{10, "0010_guard_close.sql"},
}
