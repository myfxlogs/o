-- fin_0241 OSS 三族骨架评估判据键播种（工单O-35489——判据机读配置
-- 种在 fin 链惯例同 fin_0195 research.dirtrend.*）：
-- 全值=假想卡#578380-82 evidence.falsify/ev_formula/requirements 可算
-- 口径直引，禁虚构常数（卡未载参数不播种——评估臂落地时按卡语义补）。
--
-- xsec-perp-mom-basket（卡#578380：ev_formula=xsec mom premium
--   gross20-40%/yr - 4legs x12 x7.2bp~3.5%/yr => net>15%/yr；
--   falsify=90d paper Sharpe<1 or gross<12%/yr；
--   requirements=rebal monthly, friction 7.2bp RT；
--   evidence.mut=basket k3）：
--   friction_rt=0.00072（7.2bp RT 卡载）；rebal_per_year=12（月频）；
--   basket_k=3（卡 mut 载 k3）；paper_window_d=90；
--   falsify_sharpe_lb=1.0 / falsify_gross_annual_min=0.12 /
--   target_net_annual=0.15（判词三段式=证伪界+目标界分键）。
-- funding-extreme-fade（卡#578381：|funding|>p95 fade；
--   ev_formula=gross10-25%/yr - 30trades x7.2bp~2.2% => net>7%/yr；
--   falsify=paper90d win<55% or net<5%/yr）：
--   funding_pctile_gate=0.95；est_trades_per_year=30；friction_rt=0.00072；
--   paper_window_d=90；falsify_winrate_min=0.55 / falsify_net_annual_min=0.05 /
--   target_net_annual=0.07。
-- mark-last-disloc-fade（卡#578382：mark-last deviation MR 2-5bp/event；
--   7.2bpRT>edge unless maker-only or >=10bp disloc gate；
--   falsify=event_rate x net_edge <0 => kill——结构判词无数值键）：
--   disloc_gate_bp=10（卡载备选闸）；edge_bp_min=2 / edge_bp_max=5
--   （MR 2-5bp/event 区间）；friction_rt=0.00072。

INSERT INTO app_config(key,value) VALUES
  ('research.xsec_mom.friction_rt','0.00072'),
  ('research.xsec_mom.rebal_per_year','12'),
  ('research.xsec_mom.basket_k','3'),
  ('research.xsec_mom.paper_window_d','90'),
  ('research.xsec_mom.falsify_sharpe_lb','1.0'),
  ('research.xsec_mom.falsify_gross_annual_min','0.12'),
  ('research.xsec_mom.target_net_annual','0.15'),
  ('research.funding_fade.funding_pctile_gate','0.95'),
  ('research.funding_fade.friction_rt','0.00072'),
  ('research.funding_fade.est_trades_per_year','30'),
  ('research.funding_fade.paper_window_d','90'),
  ('research.funding_fade.falsify_winrate_min','0.55'),
  ('research.funding_fade.falsify_net_annual_min','0.05'),
  ('research.funding_fade.target_net_annual','0.07'),
  ('research.disloc_fade.disloc_gate_bp','10'),
  ('research.disloc_fade.edge_bp_min','2'),
  ('research.disloc_fade.edge_bp_max','5'),
  ('research.disloc_fade.friction_rt','0.00072')
ON CONFLICT (key) DO NOTHING;
