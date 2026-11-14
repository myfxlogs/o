-- 1095 方向族首发面 md 侧（工单 trend_follow 假想卡采纳三件；fin 侧=
-- fin_0176 同构——同一发布 boot 内双侧齐落，防镜像 eval/proposal 携
-- long_perp/short_perp 落对侧即 CHECK 拒=批毒丸重试环，0083/1074 同型前案）：
-- ① md 侧 fin_opp_eval/fin_proposal direction 词表扩同集；
-- ② fin_opp_template 登记 dir_perp_mom（模板册正本在 md——fin_0052 分链；
--   params=腿原语+信号参数快照，判据真值在 fin 链 app_config）；
-- ③ 假想卡采纳落账（fin_opp_hypothesis state='adopted'——批准径显式件，
--   卡非执行单：执行面=评估臂+execfam 册+paper 记账三件代码侧见
--   internal/research/dirmom 与 internal/portfolio/lifecycle/dirpaper.go）。
-- 纸面域裁定同 fin_0176：dir_perp_mom=paper-only（真单径另棒，Drillable=false
-- 拦 probe→live）。

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

INSERT INTO fin_opp_template(template,venue,surface,params) VALUES
  ('dir_perp_mom','okx','perp_directional',
   '{"legs":["perp_long","perp_short"],"signal":"mom_z","settle":"mtm"}')
ON CONFLICT DO NOTHING;

INSERT INTO fin_opp_hypothesis(venue,surface,family,leg_primitives,requirements,state,evidence)
VALUES ('okx','perp_directional','directional',
  '["perp_long","perp_short"]',
  '{"evaluator":"动量 z 分评估（md_klines 1D 确认棒）","risk_thresholds":["z_entry","hold_days","min_net_apr"],"data_needs":["md_klines 1D","md_ticks mark"]}',
  'adopted',
  '{"provenance":"棒1 方向族采纳（方向性 edge 验证第一性——每仓=一次可证伪信念，校准账直接消费 pred vs real）","paper_only":true}')
ON CONFLICT (venue,surface) DO UPDATE SET state='adopted',evidence=EXCLUDED.evidence;
