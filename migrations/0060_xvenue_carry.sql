-- 0060 T-0144 跨所 carry 全链种：信号臂亮牌阈+黑名单表幂等保底（同构 0059——
-- 合并序无冲突；T-0144 分支依赖 T-0142 未合时的自洽面）。
CREATE TABLE IF NOT EXISTS fin_venue_untradeable (
    inst_id           text PRIMARY KEY,
    code              text NOT NULL,
    msg               text NOT NULL DEFAULT '',
    first_rejected_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE fin_venue_untradeable IS
  'venue 拒过的标的黑名单（T-0142/0059 同构幂等）——订阅/评估/起草/开仓/跨所评估滤除';
INSERT INTO app_config(key,value) VALUES
  ('research.signal.xvenue_carry_diff_bps', '300')
ON CONFLICT (key) DO NOTHING;

-- 跨所向入 direction 枚举闭集（fin_opp_eval 0052 闭集扩——提案径同源）。
ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_direction_check;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd'));
ALTER TABLE fin_proposal DROP CONSTRAINT IF EXISTS fin_proposal_direction_check;
ALTER TABLE fin_proposal ADD CONSTRAINT fin_proposal_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd'));

-- paper fill kind 扩（swap 入账快照——T-0144 票面钦定：拍时锁 mt5_swap meta raw 防回改）。
ALTER TABLE fin_paper_fill DROP CONSTRAINT IF EXISTS fin_paper_fill_kind_check;
ALTER TABLE fin_paper_fill ADD CONSTRAINT fin_paper_fill_kind_check
  CHECK (kind IN ('open','funding','fee','close','swap_snapshot'));
