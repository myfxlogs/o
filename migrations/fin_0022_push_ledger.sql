-- 0022 推送留痕账（T-0032 推送面三修——业主纪律 2026-10-24：报表期界一推/日历日推）。
-- fin_push_ledger=推送幂等锚：scope+key 主键，推成即记；重启查账不重推已推期
-- （报表 scope=report key=period:period_start；日历 scope=macro_daily key=date）。
CREATE TABLE IF NOT EXISTS fin_push_ledger (
    scope      text NOT NULL,
    key        text NOT NULL,
    pushed_at  timestamptz NOT NULL DEFAULT now(),
    msg_id     text NOT NULL DEFAULT '',
    PRIMARY KEY (scope, key)
);
