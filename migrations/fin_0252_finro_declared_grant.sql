-- fin_0252 法证宣告面 o_fin_ro 读权补授（审计升件#39084——grant-base-forget-view
-- 判例第二发）：fin_0106 o_fin_ro=SELECT-only 白名单窄读角色（业主面减负替代包），
-- 新建宣告面/底面漏携读授=只读席对法证对拍面自盲。本迁补齐 7 面：
--   fin_report/fin_pnl_net/fin_park_baseline/fin_known_residual/fin_exec_drill/
--   fin_guard_close/fin_strategy_transition（成分劈①主面+豁免册+演习凭据+关停判词）。
-- 授式同 fin_0170/0226/0229 判例（角色存在才授）；check_bridge_registry §①c
-- 同迁钉入绿链——部署窗内未迁移=grace 跳，已迁移缺权=红（授权回潮 CI 拦）。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
		EXECUTE 'GRANT SELECT ON fin_report, fin_pnl_net, fin_park_baseline,
			fin_known_residual, fin_exec_drill, fin_guard_close,
			fin_strategy_transition TO o_fin_ro';
	END IF;
END $$;
