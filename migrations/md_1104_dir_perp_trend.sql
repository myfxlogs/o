-- 1104 direction 词表扩 'perp_trend'（工单O-35491②/源票#35487——
-- fin 侧对偶件=fin_0240 同改：trendmech 评估臂 dir="perp_trend"
-- 撞 fin_opp_eval_direction_check 17 词闸=评估卡零落；语义真值
-- 扩集不改义。fin_opp_eval+fin_proposal 两表同改防镜像（0083/
-- 1074/1095/1100 同型前案；两链 ALTER 归一化全等断言⑥在案）。
-- 两段式（#25799⑩）：DROP→ADD NOT VALID→VALIDATE 分离扫。

ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_direction_check;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread','stake_carry',
                       'long_perp','short_perp','grid_spot','perp_trend')) NOT VALID;
ALTER TABLE fin_proposal DROP CONSTRAINT IF EXISTS fin_proposal_direction_check;
ALTER TABLE fin_proposal ADD CONSTRAINT fin_proposal_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread','stake_carry',
                       'long_perp','short_perp','grid_spot','perp_trend')) NOT VALID;

ALTER TABLE fin_opp_eval VALIDATE CONSTRAINT fin_opp_eval_direction_check;
ALTER TABLE fin_proposal VALIDATE CONSTRAINT fin_proposal_direction_check;
