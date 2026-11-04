-- fin_0107 T-0322 件B：COS 秘书席在册行——纯交互席（O-DM pane=席本体，
-- 不设无头形态，业主令 2026-11-04 二次修订）。无 daemon 自注册——
-- 编制册落运行花名册走迁移件（fin_0041 直改先例）；活性以 ops_event
-- 行证非 Beat（纯交互无心跳径）。
INSERT INTO fin_employee_registry(name,dept,title,status,meta) VALUES
    ('COS','platform','COS 秘书席——业主办公室（准备决策不代拍板）','active',
     '{"form":"pure_interactive","pane":"O-DM","skill":"skills/COS"}'::jsonb)
ON CONFLICT(name) DO UPDATE SET
    dept=EXCLUDED.dept, title=EXCLUDED.title,
    status=EXCLUDED.status, meta=EXCLUDED.meta;
