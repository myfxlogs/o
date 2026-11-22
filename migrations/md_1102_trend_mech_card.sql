-- md_1102: 假想卡② trend_mech 落账（工单O-32604——crypto 机械趋势
-- 手术刀卡，业主钦定 NN+LLM 讨论裁决：方向活路只认机械趋势一件）。
-- 卡留 proposed 归方向族死刑审查裁（spec 钦定禁直置 adopted）——
-- paper 评估径=internal/research/trendmech 臂产 fin_opp_eval 证据行
-- （全窗重放 Donchian20 入/10 反向出，t 收盘信号→t+1 开盘成交）。
INSERT INTO fin_opp_hypothesis(venue,surface,family,leg_primitives,requirements,state,evidence)
VALUES ('okx','trend_mech','directional',
  '["perp_long","perp_short"]',
  '{"data":"md_klines 日级确认棒+md_funding 结算行","risk":"趋势反转回撤+震荡磨损（通道 whip-saw）","evaluator":"paper窗日终净值曲线×费后净期望（fin_opp_eval trend_mech 卡 evidence.nav_curve）","prereq":"方向族豁免裁（本卡=该裁的输入件——工单#25524 死刑审查）","rule":"BTC-USDT-SWAP 1D：收盘破前20日高开多/破前20日低开空，10日反向突破平仓；t收盘信号→t+1开盘成交","notional":"perp 单腿 1x notional 上限"}',
  'proposed',
  '{"provenance":"工单O-32604 施工席假想卡②（手术刀=单范式单参数 N=20/N=10 禁扫参）","paper_only":true,"arm":"internal/research/trendmech","falsify":"paper窗≥30d——net_apr≤费调后零→判死引方向死刑审查并档;>0→升confGate排队(仍无live权)"}')
ON CONFLICT (venue,surface) DO NOTHING;
