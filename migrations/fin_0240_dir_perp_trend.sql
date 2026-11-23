-- 0240 direction 词表扩 'perp_trend'（工单O-35491②/源票#35487：
-- trendmech 评估臂 run.go dir="perp_trend" 写 fin_opp_eval 撞
-- fin_opp_eval_direction_check 17 词闸=评估卡零落；语义真值扩集
-- 不改义——perp_trend=perp-trend-daily 模板族方向词（md_1096/
-- fin_0195 同族在册），词表=md_1100/fin_0215 全量照抄+perp_trend
-- 追加；fin_opp_eval+fin_proposal 两表同改防镜像（eval/proposal
-- 携新词落对侧即 CHECK 拒=批毒丸重试环，0083/1074/1095/1100 同型
-- 前案；md 侧对偶件=md_1104 同改——两链 ALTER 归一化全等）。
-- 两段式（#25799⑩）：DROP 旧约束→ADD NOT VALID 即时挂（零存量扫）
-- →VALIDATE 分离扫转正——扩集单向放宽存量行必过。

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
