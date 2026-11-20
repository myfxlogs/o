-- 1100 网格评估族 gridfit md 侧（工单#25737 业主钦点族——方向互补：
-- 震荡市收割，方向族同市弱构实证；本件只建评估面不建执行面）：
-- ① md 侧 fin_opp_eval/fin_proposal direction 词表扩 'grid_spot'（词表=
--   md_1095/fin_0176 全量照抄+grid_spot 追加——两链同改防镜像 eval/
--   proposal 携 grid_spot 落对侧即 CHECK 拒=批毒丸重试环，0083/1074/
--   1095 同型前案；fin 侧=fin_0215 对偶件）；
-- ② fin_opp_template 登记 grid_spot（模板册正本在 md——fin_0052 分链；
--   params=腿原语+信号参数快照，判据真值在 fin 链 app_config）。
-- 影子语义同 mt5_dir_mom（T-0408 先例）：execfam 不册 grid_spot——
-- writeEval T-0330 同闸全落 rejected/exec_family_missing，eval 卡
-- （band/net_apr/realized 证据面）全可见、fin_proposal 零到达=评分先行
-- 执行面零开通；执行面=后续工单另件。

ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_direction_check;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread','stake_carry',
                       'long_perp','short_perp','grid_spot'));
ALTER TABLE fin_proposal DROP CONSTRAINT IF EXISTS fin_proposal_direction_check;
ALTER TABLE fin_proposal ADD CONSTRAINT fin_proposal_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread','stake_carry',
                       'long_perp','short_perp','grid_spot'));

INSERT INTO fin_opp_template(template,venue,surface,params) VALUES
  ('grid_spot','okx','spot_grid',
   '{"legs":["spot_long"],"signal":"grid_band_sim","settle":"mtm","shadow":true}')
ON CONFLICT DO NOTHING;
