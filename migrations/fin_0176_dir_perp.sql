-- 0176 方向族首发面（工单 trend_follow 假想卡采纳三件之①②；md 研究侧=
-- md_1095 同构——同一发布 boot 内双侧齐落，见 md_1095 头注）：
-- ① fin_opp_eval/fin_proposal direction 词表扩 long_perp/short_perp
--   （单腿方向位——perp-only 无对冲腿，方向=纯多/纯空）；
-- ② fin_paper_fill kind 词表扩 'pnl'（方向面 MTM 摊提笔——收入流观测粒度，
--   confGate 序列/晋升净额同消费）；
-- ③ 评估臂判据键播种（research.dirmom.*——判据机读配置种在 fin 链惯例）。
-- 纸面域裁定：dir_perp_mom=paper-only 族（execfam.PaperOnly 册载——excell
-- 免演登记非静默漏；真单径=executor+单腿演习件另棒接通，Drillable=false
-- 拦 probe→live 在位）。

ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_direction_check;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread','stake_carry',
                       'long_perp','short_perp'));
ALTER TABLE fin_proposal DROP CONSTRAINT IF EXISTS fin_proposal_direction_check;
ALTER TABLE fin_proposal ADD CONSTRAINT fin_proposal_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread','stake_carry',
                       'long_perp','short_perp'));

ALTER TABLE fin_paper_fill DROP CONSTRAINT IF EXISTS fin_paper_fill_kind_check;
ALTER TABLE fin_paper_fill ADD CONSTRAINT fin_paper_fill_kind_check
  CHECK (kind IN ('open','funding','fee','close','swap_snapshot','swap','pnl'));

-- 评估臂判据（缺=臂不起草 fail-closed——值域见 internal/research/dirmom）：
INSERT INTO app_config(key,value) VALUES
  ('research.dirmom.scan_interval_min','240'),
  ('research.dirmom.lookback_days','20'),
  ('research.dirmom.z_entry','1.5'),
  ('research.dirmom.hold_days','10'),
  ('research.dirmom.persist','0.5'),
  ('research.dirmom.friction_rt','0.001'),
  ('research.dirmom.min_net_apr','0.05')
ON CONFLICT (key) DO NOTHING;
