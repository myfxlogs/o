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
	{9, "0009_execution.sql"},
	{10, "0010_ledger.sql"},
	{11, "0011_review.sql"},
	{12, "0012_guard_close.sql"},
	{13, "0013_holding.sql"},
	{14, "0014_strategy_dedupe.sql"},
	{15, "0015_earn.sql"},
	{16, "0016_earn_rate.sql"},
	{17, "0017_report.sql"},
	{18, "0018_dd_hardline.sql"},
	{19, "0019_feishu_notify.sql"},
	{20, "0020_macro_event.sql"},
}
