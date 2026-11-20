-- 0215 网格评估族 gridfit fin 侧（工单#25737——md 侧=md_1100 对偶件，
-- 同一发布 boot 内双侧齐落见 md_1100 头注）：
-- ① fin_opp_eval/fin_proposal direction 词表扩 'grid_spot'（md_1095/
--   fin_0176 全量照抄+grid_spot——八桥同形断言两链 ALTER 归一化全等）；
-- ② 评估臂判据键播种（research.gridfit.* 九键——判据机读配置种在 fin
--   链惯例同 fin_0176/fin_0195；stale_h=56 钦定入册——dirmom 经
--   fin_0180 播种先例，dirtrend 懒读不播先例并存，SSOT⑦ 钉值照落）。
--   band_pct=网格带半宽；step_pct=买线步距/卖价加成；line_notional_usd=
--   每档名义额（capital=档数×名义）；fee_rt=单边成交费率（买卖各计）；
--   min_net_apr=净 APR 下界；blown_tol=跌穿带底容差（last<band_lo×
--   (1-blown_tol)=band_blown 拒）；scan_interval_min=评估拍距；
--   lookback_days=回放窗（取 lookback+1 确认棒：首棒锚 c0+lookback 仿真棒）。

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

INSERT INTO app_config(key,value) VALUES
  ('research.gridfit.scan_interval_min','240'),
  ('research.gridfit.lookback_days','90'),
  ('research.gridfit.band_pct','0.15'),
  ('research.gridfit.step_pct','0.01'),
  ('research.gridfit.line_notional_usd','50'),
  ('research.gridfit.fee_rt','0.001'),
  ('research.gridfit.min_net_apr','0.05'),
  ('research.gridfit.blown_tol','0.05'),
  ('research.gridfit.stale_h','56')
ON CONFLICT (key) DO NOTHING;
