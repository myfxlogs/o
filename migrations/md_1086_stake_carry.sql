-- 1086 T-0320 双头 carry 发现层校准 md 链：
--   ① fin_opp_eval/fin_proposal direction 闭集扩 'stake_carry'（桥表 DDL 与
--      fin 链 fin_0105 同形——归一化断言钉）；
--   ② fin_capability_surface 资产质押维（左元 product_family='staking'：
--      stakable 旗+rate_source 率源字段）+空单付费 venue 维（右元
--      product_family='swap-carry'——「TRX 空单 swap 恒正」对发现层可见化）；
--      二维积=stakable×swap-carry on ccy join=假想官 stakeCarryScan 枚举对象；
--   ③ mt5 TRX 面实测登记（EXR 3001157 探针 2026-11-04：TRXUSD 在目录但
--      SwapType=SwapNone+TradeMode=Disabled——venue 覆盖缺口如实关面登记）。

ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_direction_check;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread','stake_carry'));
ALTER TABLE fin_proposal DROP CONSTRAINT IF EXISTS fin_proposal_direction_check;
ALTER TABLE fin_proposal ADD CONSTRAINT fin_proposal_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread','stake_carry'));

INSERT INTO fin_capability_surface(venue,surface,endpoint_family,product_family,rw_class,cost_attrs,enabled) VALUES
-- 资产质押维（左元）：TRX 链上质押——率源=链上实收口径（vote+rental）。
('tron','staking-TRX','tron-chain-staking','staking','read',
 '{"ccy":"TRX","stakable":true,"rate_source":"trx_onchain","note":"链上质押=能量租金+投票——T-0320 双头 carry 标定"}',true),
-- 空单付费 venue 维（右元）：CMC TRX 空单隔夜结算恒正（业主实盘实测）；
-- OKX TRX-USDT-SWAP 空 perp funding 面；MT5 TRXUSD 实测零 carry 关面登记。
('cmc','swap-carry-TRX','cmc-rest:history','swap-carry','read',
 '{"ccy":"TRX","instrument":"X-AFQUL","direction":"short","rate_source":"cmc_realized","note":"TRX 空单隔夜结算恒正（业主实盘实测）——T-0320"}',true),
('okx','swap-carry-TRX','ws-public:funding-rate','swap-carry','read',
 '{"ccy":"TRX","inst":"TRX-USDT-SWAP","direction":"short","rate_source":"funding_view"}',true),
('mt5','swap-carry-TRX','mtapi:SymbolParams','swap-carry','read',
 '{"ccy":"TRX","symbol":"TRXUSD","direction":"short","rate_source":"mt5_swap","note":"EXR 3001157 实测 SwapNone+TradeMode_Disabled（2026-11-04 T-0320 探针）——venue 覆盖缺口如实登记"}',false)
ON CONFLICT (venue,surface) DO NOTHING;
