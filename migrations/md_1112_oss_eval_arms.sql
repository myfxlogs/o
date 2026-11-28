-- md_1112 OSS 三族骨架评估臂 md 侧（工单O-41394——fin 侧对偶件
-- =fin_0278 同改：direction 词表两链同改防镜像 eval/proposal 携
-- perp_fade/xsec_mom_basket 落对侧即 CHECK 拒=批毒丸重试环，
-- 0270/1111/0240/1104 同型前案；两链 ALTER 归一化全等断言在案）：
-- ① fin_opp_eval/fin_proposal direction 词表扩 'perp_fade'/'xsec_mom_basket'
--   （=grammar.go 三族 Direction 注册值；19→21 词，与 fin_0278 全等）。
-- ② 判据键真值种在 fin 链 app_config（fin_0278/fin_0243）——SG 侧
--   app_config 副本经 replica 径同步，本迁移不播种键。
--   两段式（#25799⑩）：DROP→ADD NOT VALID→VALIDATE 分离扫。

ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_direction_check;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread','stake_carry',
                       'long_perp','short_perp','grid_spot','perp_trend','grid_cfd',
                       'perp_fade','xsec_mom_basket')) NOT VALID;
ALTER TABLE fin_proposal DROP CONSTRAINT IF EXISTS fin_proposal_direction_check;
ALTER TABLE fin_proposal ADD CONSTRAINT fin_proposal_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread','stake_carry',
                       'long_perp','short_perp','grid_spot','perp_trend','grid_cfd',
                       'perp_fade','xsec_mom_basket')) NOT VALID;

ALTER TABLE fin_opp_eval VALIDATE CONSTRAINT fin_opp_eval_direction_check;
ALTER TABLE fin_proposal VALIDATE CONSTRAINT fin_proposal_direction_check;
