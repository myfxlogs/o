-- fin_0091 每席预算账（迁移号并合时 0090→0091 避 trx_rental_payers 撞车）（T-0267 件A——业主准两件入计划 MANAGE-245）。
-- ops_ticket_usage=append-only 账行：每次无头 spawn（施工 run/审计 audit）退出时
-- 由 meta 侧 headless-usage.sh 解析 devin --export ATIF final_metrics 落一行；
-- token 列 NULL=不可得（DRYRUN 桩/无 --export 径 spawn——「不可得」如实记不造数）。
-- 写径=FIN_PG_DSN 127.0.0.1:5544 research_rw（report.sh ops_event 同源）；
-- 读径=o-core GetCockpit 工单板成本列（票→累计 token 聚合 LEFT JOIN）。
CREATE TABLE IF NOT EXISTS ops_ticket_usage (
    id                bigserial PRIMARY KEY,
    ts                timestamptz NOT NULL DEFAULT now(),
    ticket            text NOT NULL,                    -- T-NNNN 票号
    kind              text NOT NULL CHECK (kind IN ('run','audit')),  -- spawn 型域
    model             text NOT NULL DEFAULT '',         -- 底模（ATIF agent.model_name 实读）
    prompt_tokens     bigint,                           -- 入 token（NULL=不可得）
    completion_tokens bigint,                           -- 出 token（NULL=不可得）
    cached_tokens     bigint,                           -- 缓存命中 token（NULL=不可得）
    source            text NOT NULL DEFAULT 'atif',     -- 数据源域（atif=实测/none=不可得）
    note              text NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS ops_ticket_usage_ticket ON ops_ticket_usage(ticket, ts DESC);
COMMENT ON TABLE ops_ticket_usage IS
  '每席预算账（T-0267）——无头单 token 用量 append-only 账行；写=meta wrapper headless-usage.sh（FIN_PG_DSN research_rw），读=o-core GetCockpit 工单板成本列';

-- research_rw 写径授权（report.sh ops_event 同族规——角色缺席环境=零失败）。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'research_rw') THEN
    GRANT INSERT ON ops_ticket_usage TO research_rw;
    GRANT USAGE ON SEQUENCE ops_ticket_usage_id_seq TO research_rw;
    GRANT SELECT ON ops_ticket_usage TO research_rw;
  END IF;
END $$;
