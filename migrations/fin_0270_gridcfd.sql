-- 0270 CFD 网格评估影子族 grid_cfd fin 侧（工单O-41369——md 侧对偶件
-- =md_1111 同改：direction 词表两链同改防镜像 eval/proposal 携
-- grid_cfd 落对侧即 CHECK 拒=批毒丸重试环，0083/1074/1095/1100/1104
-- 同型前案；两链 ALTER 归一化全等断言⑥在案）：
-- ① fin_opp_eval/fin_proposal direction 词表扩 'grid_cfd'（md_1111/
--   fin_0240 全量照抄+grid_cfd——18 词）；
-- ② 评估臂判据键播种（research.gridcfd.* 九键——判据机读配置种在 fin
--   链惯例同 fin_0215/fin_0240；stale_h=56 同 gridfit 钉值懒读兜底）。
--   band_pct=网格带半宽（缺省 0.10——FX majors 90d 实幅约 3-8%、
--   股指/金属略宽，10% 带覆盖多数标的又不致死仓过宽；exotic 自拒
--   属预期）；step_pct=买线步距/卖价加成（缺省 0.005≈FX 日 vol
--   ~0.5% 一倍——相邻档约日频触发，价差成本 cost_rt=spread 中位
--   对拍后净利仍正才有意义）；line_notional_usd=每档名义额；
--   fee_floor=cost_rt 下界（缺省 0.002=0.2%——BTCUSDm 级紧价差锚
--   钦值；实测 spread>floor 用实测——majors 实测 ~0.005-0.01% 由
--   floor 主导、exotic 实测 5-15% 如实拒）；min_net_apr=净 APR
--   下界；blown_tol=跌穿带底容差；scan_interval_min=评估拍距；
--   lookback_days=回放窗（取 lookback+1 确认棒：首棒锚 c0+lookback
--   仿真棒——exness 1D 史 202d≥90d 在案）。
--   两段式（#25799⑩）：DROP→ADD NOT VALID→VALIDATE 分离扫。

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

INSERT INTO app_config(key,value) VALUES
  ('research.gridcfd.scan_interval_min','240'),
  ('research.gridcfd.lookback_days','90'),
  ('research.gridcfd.band_pct','0.10'),
  ('research.gridcfd.step_pct','0.005'),
  ('research.gridcfd.line_notional_usd','50'),
  ('research.gridcfd.fee_floor','0.002'),
  ('research.gridcfd.min_net_apr','0.05'),
  ('research.gridcfd.blown_tol','0.05'),
  ('research.gridcfd.stale_h','56')
ON CONFLICT (key) DO NOTHING;
