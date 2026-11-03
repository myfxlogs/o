-- 0097 T-0298 基差族判据键种＋启封台账 ops_event（D-0043/CTO 会签）：
--   ① research.basis.min_net_apr=0.06（与 T-0297 funding 静态底同值归位）；
--   ② research.basis.min_days_to_expiry=7（到期<7d=滚动/交割风险早退拒）；
--   ③ fin_opp_template 两模板启封属 SG 数据面（md 链 md_1080 行 UPDATE）——
--     ops_event 留痕落本 fin 链（ops_event 归 fin 域册）。
INSERT INTO app_config(key,value) VALUES
  ('research.basis.min_net_apr',        '0.06'),
  ('research.basis.min_days_to_expiry', '7')
ON CONFLICT (key) DO NOTHING;

INSERT INTO ops_event(seat,event,ticket,note) VALUES
  ('施工席','启用','T-0298',
   'fin_opp_template enabled=true+state=enabled×2（quarter_basis_long/perp_quarter_spread）台账归位——D-0043/T-0287；quarter_basis_short 保禁（spot_short 腿缺 margin 能力面+不在 paperTemplates）');
