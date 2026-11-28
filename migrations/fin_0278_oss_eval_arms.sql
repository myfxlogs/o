-- 0278 OSS 三族骨架评估臂 fin 侧（工单O-41394——md 侧对偶件=md_1112
-- 同改：direction 词表两链同改防镜像 eval/proposal 携新词落对侧即
-- CHECK 拒=批毒丸重试环，0270/0240/1104 同型前案；两链 ALTER 归一化
-- 全等断言在案）：
-- ① fin_opp_eval/fin_proposal direction 词表扩 'perp_fade'/'xsec_mom_basket'
--   （=grammar.go 三族 Direction 注册值——funding-extreme-fade/
--   mark-last-disloc-fade 同 perp_fade，xsec-perp-mom-basket 独值；
--   19→21 词）；
-- ② 评估臂判据键播种——fin_0243 已种卡面判据键（pctile/edge/friction/
--   target/falsify），本迁移只补臂运行键：scan_interval_min/stale 界/
--   periods_per_year/stale_min/lookback_days。xsec_mom.lookback_days=30
--   按 fin_0243 注记「卡未载参数不播种——评估臂落地时按卡语义补」
--   授权补键（月频 rebal 对齐的标准动量窗）；stale_h=56 同
--   gridfit/gridcfd 钉值（1D 确认棒滞后+日 cadence 常龄≤40h，56h 顶
--   =真断供仍捕陈）；funding stale_h=26（8h 结算间隔+slack）；
--   disloc stale_min=30（state 表 recv_ts 鲜度界）。
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

INSERT INTO app_config(key,value) VALUES
  ('research.funding_fade.scan_interval_min','360'),
  ('research.funding_fade.stale_h','26'),
  ('research.funding_fade.periods_per_year','1095'),
  ('research.disloc_fade.scan_interval_min','240'),
  ('research.disloc_fade.stale_min','30'),
  ('research.xsec_mom.scan_interval_min','360'),
  ('research.xsec_mom.lookback_days','30'),
  ('research.xsec_mom.stale_h','56')
ON CONFLICT (key) DO NOTHING;
