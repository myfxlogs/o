-- 0063 fin_data_retention 表级保留账（T-0153——保留窗裁定数据面化：逐表
-- 在册=清道夫照账执行；新表不入账=报缺 fail-closed）。
-- class: 'audit_keep'=可删行账（keep_interval 生效）；'permanent'=账链永久
-- 禁删；'state'=状态表不删行（upsert 键域封顶）。
CREATE TABLE IF NOT EXISTS fin_data_retention (
    table_name    text PRIMARY KEY,
    class         text NOT NULL CHECK (class IN ('audit_keep','permanent','state')),
    keep_interval interval,
    last_swept_at timestamptz,
    size_bytes    bigint,
    note          text
);
-- 全表默认入册（防新表漂出账面——deploy 时快照在册，后续新表须显式批注）。
-- current_schema() 非 'public' 钉死——testutil 包级 schema 隔离（T-0021）下
-- 迁移表落测试 schema，钉 public 快照=空账（清道夫 fail-closed 全拒动）。
INSERT INTO fin_data_retention(table_name,class,note)
SELECT t.tablename,'permanent','默认账链保留——新表须显式裁窗'
FROM pg_tables t WHERE schemaname=current_schema()
ON CONFLICT (table_name) DO NOTHING;
-- 钦定初值覆写（T-0153 票面钦定）。
UPDATE fin_data_retention SET class='audit_keep',keep_interval=interval '30 days',
  note='judged 终局行 30d 清窗' WHERE table_name='fin_forward_check';
UPDATE fin_data_retention SET class='audit_keep',keep_interval=interval '90 days',
  note='所属策略退役后 90d 清窗' WHERE table_name='fin_paper_fill';
UPDATE fin_data_retention SET class='state',
  note='状态表不删行——upsert 键域封顶（churn 死元组走 avac 参数面）'
  WHERE table_name IN ('md_ticks','md_mark_index','md_funding','md_account_events',
  'md_earn_rate','md_dcd_product','fin_opp_eval','fin_opp_hypothesis',
  'fin_capability_surface','fin_surface_gap','app_config','app_secrets',
  'fin_strategy','fin_proposal','fin_opp_template','fin_condition_gap',
  'fin_data_retention','migrate_history','fin_employee_registry');
UPDATE fin_data_retention SET keep_interval=interval '30 days'
  WHERE table_name='md_funding'; -- 既有 timescale policy 不动（票面钦定）
UPDATE fin_data_retention SET keep_interval=interval '90 days'
  WHERE table_name='md_equity_curve'; -- 同上——policy 管删非清道夫
-- A 类热表 autovacuum 表级参数调激进（死元组追不上写压实证——堆膨胀平台期
-- 治理：scale_factor 0.02/cost_delay 2ms）。
ALTER TABLE fin_opp_eval SET (autovacuum_vacuum_scale_factor=0.02, autovacuum_vacuum_cost_delay=2);
ALTER TABLE md_mark_index SET (autovacuum_vacuum_scale_factor=0.02, autovacuum_vacuum_cost_delay=2);
ALTER TABLE md_ticks SET (autovacuum_vacuum_scale_factor=0.02, autovacuum_vacuum_cost_delay=2);
