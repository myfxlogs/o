-- 0244 构性残差豁免跟踪化（工单O-36327——业主令④「豁免续命即失职」根修）：
-- fin_known_residual 静态定额豁免对活值残差恒滞后（DCD口径差随BTC漂/
-- Earn应计逐日增生），每爬过豁口告警一次、人工+$1续豁三连实证(#20-22)。
-- 修法=加 resolver 列：构性行存解析键非死数，豁免额=静态行+跟踪行现算
-- （internal/ledger/residual_tracked.go 注册表；NULL=纯静态向后兼容）。
--
-- 覆盖切换：#18(DCD市值漂移$53)/#19(Earn应计$10) 置 resolver 键——
-- 静态额原值留 amount_usd 列作锚值（|跟踪−锚|>max(2×锚,$25)→破带按新族
-- 异常升件不续构性豁免，检测力不稀释）；#20-22 补差行作废——同族续增生
-- 由跟踪行现算吸收，到期不再续。

ALTER TABLE fin_known_residual ADD COLUMN IF NOT EXISTS resolver text;

UPDATE fin_known_residual SET resolver='dcd_mtm_drift:11448262'
	WHERE id=18 AND resolver IS NULL;
UPDATE fin_known_residual SET resolver='earn_accrual_gap'
	WHERE id=19 AND resolver IS NULL;

UPDATE fin_known_residual
	SET expires_at=now(),
	    reason=reason||'｜作废判词:构性残差跟踪化接管(O-36327)——同族续增生由resolver现算吸收,到期不再续'
	WHERE id IN (20,21,22) AND resolver IS NULL;
