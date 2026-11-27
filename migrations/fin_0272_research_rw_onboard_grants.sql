-- fin_0272 O-43077 onboard 臂双表拒权根修（升件#43077——SQLSTATE 42501
--   实证:o-research 00:02Z 起 venue=xmglobal+exness 推进败+漂移探针
--   拍败连环;三表 owner=o,roster_rw 持 ops_event+ops_event_act 授,
--   research_rw 三表零授——onboard/survey 臂跑 research 域角色:
--   ① ops_event_act=ops_event 触发器投影基表,臂 INSERT ops_event 即
--     连带 INSERT act(同会话角色权限计)——SELECT+INSERT;
--   ② fin_symbol_dialect=DialectStore.upsert 消费面(mt5/dialect.go:
--     INSERT..ON CONFLICT DO UPDATE+SELECT 查无映射 fail-closed)——
--     SELECT+INSERT+UPDATE(DELETE 无消费面不授)。
--   授权对齐 ops_event_writepath 册规约:最小必需集,UD 锁权链
--   (fin_0121 ops_event_lockdown 角色册已含 research_rw)不动。
-- depends_on: fin_0191_ops_event_act_table.sql, fin_0135_symbol_dialect.sql
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'research_rw') THEN
		GRANT SELECT, INSERT ON ops_event_act TO research_rw;
		GRANT SELECT, INSERT, UPDATE ON fin_symbol_dialect TO research_rw;
	END IF;
END $$;
