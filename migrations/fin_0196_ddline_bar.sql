-- 回撤硬线 v2（工单#20799）：柱行偏索引+柱长键。md_equity_curve 现 760 万行/30d
-- 仅 (ts DESC) 索引——峰值查询全窗扫 5.5s；柱行（src='guard_dd_bar'，约 1 行/5 分）
-- 走偏索引毫秒级。历史 guard_dd 采点行不动（90d 保留策略自然滚出）。
CREATE INDEX IF NOT EXISTS md_equity_curve_ddbar_idx
  ON md_equity_curve (ts DESC) WHERE src='guard_dd_bar';
INSERT INTO app_config(key,value) VALUES ('risk.dd_bar_seconds','300')
ON CONFLICT (key) DO NOTHING;
