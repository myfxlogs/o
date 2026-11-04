-- 0105 T-0320 双头 carry 发现层校准 fin 链：
--   ① fin_opp_eval/fin_proposal direction 闭集扩 'stake_carry'（桥表 DDL 与
--      md 链 md_1086 同形——归一化断言钉）；
--   ② 判据配置种（热启——双头臂阈/缓冲/质押率行龄门；md 侧经 app_config
--      副本可见）。

ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_direction_check;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread','stake_carry'));
ALTER TABLE fin_proposal DROP CONSTRAINT IF EXISTS fin_proposal_direction_check;
ALTER TABLE fin_proposal ADD CONSTRAINT fin_proposal_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread','stake_carry'));

INSERT INTO app_config(key,value) VALUES
  ('research.signal.stake_carry_net_bps', '300'),
  ('research.signal.stake_carry_buffer_bps', '50'),
  ('research.stake.stale_hours', '168')
ON CONFLICT (key) DO NOTHING;
