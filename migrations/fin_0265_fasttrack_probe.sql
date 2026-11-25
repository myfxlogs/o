-- fin_0265 O-41014 快窗信号 probe 直通径阈件册（探测/测量装置——非策略扩张）：
-- eval 极值判词∧dir 族白名单→mint state='probe' 小仓跳 paper 证明窗；
-- 只越 paper 证明窗，confGate/entryGate/armed_ttl/stop-loss/dd/并发闸
-- 全继承原样。enabled 缺省 false=总闸（点亮权归部署——机制先通后亮）。
INSERT INTO app_config(key,value,updated_by) VALUES
  ('probe.fasttrack_enabled','false','fin_0265'),
  ('probe.fasttrack_score','8.0','fin_0265'),
  ('probe.fasttrack_notional_usd','50','fin_0265'),
  ('probe.fasttrack_max_open','3','fin_0265')
ON CONFLICT (key) DO NOTHING;
