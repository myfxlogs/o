-- 1111 CFD 网格评估影子族 grid_cfd md 侧（工单O-41369——fin 侧对偶
-- 件=fin_0270 同改：direction 词表两链同改防镜像 eval/proposal 携
-- grid_cfd 落对侧即 CHECK 拒=批毒丸重试环，0083/1074/1095/1100/1104
-- 同型前案；两链 ALTER 归一化全等断言⑥在案）：
-- ① md 侧 fin_opp_eval/fin_proposal direction 词表扩 'grid_cfd'
--   （md_1104/fin_0240 全量照抄+grid_cfd——18 词）；
-- ② fin_opp_template 登记 grid_cfd（模板册正本在 md——fin_0052 分链；
--   params=腿原语+信号参数快照，判据真值在 fin 链 app_config）。
-- 影子语义（exec_family_missing 闸在案）：execfam 不册 grid_cfd——
-- writeEval T-0330 同闸全落 rejected/exec_family_missing，eval 卡
-- （band/net_apr/spread_pct 对拍证据面）全可见、fin_proposal 零到达
-- =评分先行执行面零开通；执行面=后续工单另件。
-- venue='exness'（O-39440 venue 改名后真源——mt5 旧 token 不再新发）。
-- legs=["cfd_long"]——长网格库存语义（买跌档卖涨档单边多头库存，
-- 对称双腿网格=另一机制另票本票不扩）。

ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_direction_check;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread','stake_carry',
                       'long_perp','short_perp','grid_spot','perp_trend','grid_cfd')) NOT VALID;
ALTER TABLE fin_proposal DROP CONSTRAINT IF EXISTS fin_proposal_direction_check;
ALTER TABLE fin_proposal ADD CONSTRAINT fin_proposal_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread','stake_carry',
                       'long_perp','short_perp','grid_spot','perp_trend','grid_cfd')) NOT VALID;

ALTER TABLE fin_opp_eval VALIDATE CONSTRAINT fin_opp_eval_direction_check;
ALTER TABLE fin_proposal VALIDATE CONSTRAINT fin_proposal_direction_check;

INSERT INTO fin_opp_template(template,venue,surface,params) VALUES
  ('grid_cfd','exness','cfd_grid',
   '{"legs":["cfd_long"],"signal":"grid_band_sim","settle":"mtm","shadow":true}')
ON CONFLICT DO NOTHING;
