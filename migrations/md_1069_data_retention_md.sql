-- 1069 fin_data_retention PG-md 保留账（T-0169 承接段：清道夫双实例配装——
-- D-0030 §八「保留账 fin_data_retention 分库各一份」；SG 账照 fin_0063 旧，
-- PG-md 账本件独立在册（md 正本面表集）。DDL 与 fin 链 fin_0063 归一化同形
-- （桥表断言钉——分库账面同构）。清道夫-md（cmd/sweeper）照账日拍。
CREATE TABLE IF NOT EXISTS fin_data_retention (
    table_name    text PRIMARY KEY,
    class         text NOT NULL CHECK (class IN ('audit_keep','permanent','state')),
    keep_interval interval,
    last_swept_at timestamptz,
    size_bytes    bigint,
    note          text
);

-- PG-md 账面在册（md 正本全集——清道夫 fail-closed 未入账拒动，缺册即红）。
INSERT INTO fin_data_retention(table_name,class,keep_interval,note) VALUES
  ('md_ticks','state',NULL,'状态表不删行——upsert 键域封顶'),
  ('md_mark_index','state',NULL,'状态表不删行——upsert 键域封顶'),
  ('md_signal','state',NULL,'状态表不删行——upsert 键域封顶'),
  ('md_funding','state',interval '30 days','超表保留期管删（timescale policy 同 SG 口径）'),
  ('md_dcd_product','state',NULL,'状态表不删行——upsert 键域封顶'),
  ('md_earn_rate','state',NULL,'状态表不删行——upsert 键域封顶'),
  ('md_macro_event','state',NULL,'状态表不删行——第八桥正本（event_date 键）'),
  ('fin_opp_eval','state',NULL,'状态表不删行——评估正本（id 桥）'),
  ('fin_proposal','state',NULL,'状态面不删行——提案正本（退位行留痕归 SG 侧口径）'),
  ('fin_opp_template','state',NULL,'模板书不删行'),
  ('fin_opp_hypothesis','state',NULL,'假想卡不删行'),
  ('fin_capability_surface','state',NULL,'能力面注册不删行'),
  ('fin_surface_gap','state',NULL,'面差队列不删行（分类器审裁态）'),
  ('fin_template_proposal','state',NULL,'模板修正提案不删行'),
  ('fin_forward_check','audit_keep',interval '30 days','judged 终局行 30d 清窗（md 正本面——SG 账同窗）'),
  ('app_config','state',NULL,'第九桥只读副本——projector 全量 diff 维护'),
  ('fin_data_retention','state',NULL,'保留账本体不删行'),
  ('schema_migrations','permanent',NULL,'迁移基线永久')
ON CONFLICT (table_name) DO NOTHING;
