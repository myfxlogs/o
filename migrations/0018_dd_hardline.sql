-- 0018 回撤硬线执行件（T-0025 段一——业主令 2026-10-24：-3%/30d 净权益口径机械化）。
-- md_equity_curve=净权益采点事实行（append-only 统计面非态表）：源=fin_venue_account_snapshot
-- .equity（venue 自报含 uPnL——会签补正①钉死口径，纯 cashflow 已实现面被否：长持亏仓藏回撤门）。
-- 回撤语义两段（会签补正②）：史满窗口→30d 滚动窗 peak-to-now；史不足→交易起点至今 HWM。
-- 触线动作：kill_switch 落闸→逐仓 dd_30d 全平；无自动复位——翻闸只认显式热改。
CREATE TABLE IF NOT EXISTS md_equity_curve (
    ts      timestamptz NOT NULL DEFAULT now(),
    venue   text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    ccy     text NOT NULL,
    equity  numeric NOT NULL,
    src     text NOT NULL,
    recv_ts timestamptz NOT NULL DEFAULT now()
);
COMMENT ON COLUMN md_equity_curve.venue IS 'venue 域（裸值禁冒号投影）';
COMMENT ON COLUMN md_equity_curve.equity IS '净权益采点（fin_venue_account_snapshot.equity——含浮盈，对账环同基准）';
SELECT create_hypertable('md_equity_curve', 'ts', if_not_exists => TRUE);

-- risk.max_dd_30d 定值 -3% 只认业主令 2026-10-24（skills/策略评估与仓位：本卡不定阈值，
-- 调整须业主令=越权即停）；dd_window_days=滚动窗天。两键热启；缺键=不监视+告警计数。
INSERT INTO app_config(key,value) VALUES
  ('risk.max_dd_30d',    '"0.03"'),
  ('risk.dd_window_days','"30"')
ON CONFLICT (key) DO NOTHING;
