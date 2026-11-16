-- 0195 方向趋势族评估臂判据键播种（工单#11481——research.dirtrend.*
-- 八键；判据机读配置种在 fin 链惯例同 fin_0176）：
--   scan_interval_min=评估拍距；lookback_days=Donchian 通道窗+今日信号；
--   replay_days=winrate-CI 回放窗（W=lookback+replay+1 确认棒）；
--   winrate_lb=Wilson 下界线；min_signal_samples=回放样本下界；
--   hold_days=持有期（日，friction 摊销分母）；friction_rt=往返摩擦；
--   min_net_apr=净 APR 下界。
-- research.dirtrend.stale_h 懒读缺省 56h 不播种（同 dirmom 先例——
-- 代码内 56h 顶=真断供仍捕陈）；gate.kill.dirtrend 同型懒读不播种。

INSERT INTO app_config(key,value) VALUES
  ('research.dirtrend.scan_interval_min','240'),
  ('research.dirtrend.lookback_days','20'),
  ('research.dirtrend.replay_days','60'),
  ('research.dirtrend.winrate_lb','0.5'),
  ('research.dirtrend.min_signal_samples','10'),
  ('research.dirtrend.hold_days','10'),
  ('research.dirtrend.friction_rt','0.001'),
  ('research.dirtrend.min_net_apr','0.05')
ON CONFLICT (key) DO NOTHING;
