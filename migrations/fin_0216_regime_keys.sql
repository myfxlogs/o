-- 0216 regime→结构映射器判据键播种（工单#25750——research.regime.*
-- 九键；判据机读配置种在 fin 链惯例同 fin_0176/fin_0195）：
--   scan_interval_min=评估拍距；lookback_days=分类窗（取 lookback+1
--   确认棒：首棒只产锚不入仿真——z/sigma 同 dirmom 口径）；
--   z_thr=趋势阈（|z|≥→trend）；vr_thr=方差比阈（Var(r2)/(2·Var(r1))
--   <→候选 chop）；cont_thr=含带率阈（收盘含于既往通道占比≥→chop）；
--   quiet_thr/vol_thr=sigma_d 静/躁阈；thin_days=族覆盖读窗（近 N 日
--   fin_opp_eval 卡数<该值=判词稀薄）；
--   map=人维护映射表 JSON（钉值照 SSOT④——regime→候选族册，
--   映射表外族永不提案 fail-closed）。
-- research.regime.stale_h 懒读缺省 56h 不播种（同 dirtrend 先例）；
-- gate.kill.regime 同型懒读不播种。
-- fin_0215 在途占（fix/gridfit 支）——跳号钉 0216。

INSERT INTO app_config(key,value) VALUES
  ('research.regime.scan_interval_min','360'),
  ('research.regime.lookback_days','90'),
  ('research.regime.z_thr','2.0'),
  ('research.regime.vr_thr','0.9'),
  ('research.regime.cont_thr','0.6'),
  ('research.regime.quiet_thr','0.01'),
  ('research.regime.vol_thr','0.03'),
  ('research.regime.thin_days','7'),
  ('research.regime.map','{"chop":["grid_spot"],"trend":["dir_perp_mom","mt5_dir_mom"],"quiet":["vol_sell"],"volatile":[]}')
ON CONFLICT (key) DO NOTHING;
