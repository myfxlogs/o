-- 1070 fin_strategy/fin_venue_untradeable PG-md 只读副本（T-0172：研究侧 fin 域
-- 读面分域断供急诊——dedupe 闸（funding persist/dcd eval）读 fin_strategy 42P01
-- 提案管线堵；fin_venue_untradeable 起草闸黑名单三臂读败静默 fail-open 同治）。
-- SG 正本→projector 反向镜像（ev_strategy 帧 fin_0028 在册+全表 diff 同源，
-- 行量百级）；研究面只读，写权唯一=projector。DDL 与 fin 链正表归一化同形
-- （check_migrations 断言⑥ 桥/反向桥同形钉）；触发器/回填 UPDATE 不在副本。
-- 索引重放序承 0014→0050 演化（3col 建即废、4col 终态——并集库 md 先 fin 后
-- 次序幂等兼容）。fin_venue_untradeable 副本无 0061 回扫钩（正本触发器事）。
CREATE TABLE IF NOT EXISTS fin_strategy (
    id           bigserial PRIMARY KEY,
    inst_id      text NOT NULL,
    template_ref text NOT NULL,
    proposal_ref bigint,
    state        text NOT NULL CHECK (state IN ('proposed','paper','probe','live','degraded','retired')),
    notional_usd numeric NOT NULL,
    entered_at   timestamptz NOT NULL DEFAULT now(),
    criteria     jsonb NOT NULL DEFAULT '{}',
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS fin_strategy_state ON fin_strategy(state);
ALTER TABLE fin_strategy ADD COLUMN IF NOT EXISTS venue text NOT NULL DEFAULT 'okx';
DO $$ BEGIN
  ALTER TABLE fin_strategy ADD CONSTRAINT fin_strategy_venue_chk CHECK (venue NOT LIKE '%:%');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
ALTER TABLE fin_strategy ADD COLUMN IF NOT EXISTS direction text NOT NULL DEFAULT 'short_perp_long_spot';
CREATE UNIQUE INDEX IF NOT EXISTS fin_strategy_active_dedupe
    ON fin_strategy(venue, inst_id, direction) WHERE state <> 'retired';
DROP INDEX IF EXISTS fin_strategy_active_dedupe;
CREATE UNIQUE INDEX IF NOT EXISTS fin_strategy_active_dedupe
    ON fin_strategy(venue, inst_id, direction, template_ref) WHERE state <> 'retired';
COMMENT ON TABLE fin_strategy IS
  'fin_strategy PG-md 只读副本（T-0172 反向桥）——SG 正本 projector 全表 diff 维护，研究面 dedupe/考据读';

CREATE TABLE IF NOT EXISTS fin_venue_untradeable (
    inst_id           text PRIMARY KEY,
    code              text NOT NULL,
    msg               text NOT NULL DEFAULT '',
    first_rejected_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE fin_venue_untradeable IS
  'venue 拒标的黑名单只读副本（T-0172 反向桥）——SG 正本 projector diff 维护，研究面起草闸滤除';

-- 保留账入册（清道夫-md fail-closed 未入账报缺——副本=state 类不删行，同正本口径）。
INSERT INTO fin_data_retention(table_name,class,keep_interval,note) VALUES
  ('fin_strategy','state',NULL,'SG 正本反向镜像只读副本——projector diff 维护（T-0172）'),
  ('fin_venue_untradeable','state',NULL,'SG 正本反向镜像只读副本——projector diff 维护（T-0172）')
ON CONFLICT (table_name) DO NOTHING;
