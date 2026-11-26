-- fin_0267 margin-short 追踪面缺口修件（O-41350——O-40707 post-merge 三维审
-- 遗留，margin.exec_enabled 武装前置）：
-- ①flatten_failed 退出枚举=息腿停/健康闸失明/永不重试三同根——枚举窗扩+
--   重试计数/防抖时戳/升件封顶三列补齐；
-- ②unique 部分索引 WHERE 同步加 flatten_failed——同币双借封死
--   （venue 侧双负债叠加事故面）；
-- ③裸负债升件 esc_class=margin_naked_liab 新册（借成确认败/卖腿败——
--   venue 侧可能真借而台账全不知，人工可见不静默）；
-- ④flatten 败连击封顶升件 esc_class=margin_flatten_fail（防抖闸+升件
--   封顶——禁重试无界）；
-- ⑤闸键两枚：flatten_max_fails 重试封顶（缺/非正=0=不自动重试直接升件
--   fail-closed）+flatten_retry_ivl_s 防抖间隔。

ALTER TABLE fin_margin_loan
    ADD COLUMN IF NOT EXISTS flatten_fails      integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_flatten_try_at timestamptz,
    ADD COLUMN IF NOT EXISTS flatten_escalated   boolean  NOT NULL DEFAULT false;
COMMENT ON COLUMN fin_margin_loan.flatten_fails IS 'flatten 序连续败计数——达 margin.flatten_max_fails 封顶升件 margin_flatten_fail 后停自动重试（人工处置面，O-41350）';
COMMENT ON COLUMN fin_margin_loan.flatten_escalated IS '封顶升件已落标记——一次性真源防重复刷件';

-- 存量零行安全：fin_margin_loan 武装前空表（exec_enabled=false 未开仓），
-- 索引重建无存量违例面。
DROP INDEX IF EXISTS fin_margin_loan_one_open;
-- mig-rehearsal-ok: fin_margin_loan 武装前空集（exec_enabled=false 未开仓
-- 零 loan 行）——唯一索引重建无存量违例面；即便有行，WHERE 扩态只收紧
-- 同币唯一约束，生产侧双态并存本就属事故面应拒。
CREATE UNIQUE INDEX fin_margin_loan_one_open
    ON fin_margin_loan (venue, ccy)
    WHERE status IN ('open','closing','flatten_failed');
DROP INDEX IF EXISTS fin_margin_loan_open;
CREATE INDEX fin_margin_loan_open
    ON fin_margin_loan (status)
    WHERE status IN ('open','closing','flatten_failed');

INSERT INTO app_config(key,value) VALUES
  ('margin.flatten_max_fails','8'),
  ('margin.flatten_retry_ivl_s','600')
ON CONFLICT (key) DO NOTHING;

INSERT INTO ops_esc_valence(esc_key,weight,note) VALUES
  ('margin_naked_liab',85,'margin-short 借成未簿记裸负债——venue 侧可能真借而台账/闸全不知，须人工对账（工单O-41350）'),
  ('margin_flatten_fail',75,'margin-short flatten 序连续败封顶——负债真实在账但自动平仓序已停，须人工处置（工单O-41350）')
ON CONFLICT (esc_key) DO NOTHING;
