-- 0039 预测费率列（T-0084：funding-rate 帧 nextFundingRate 采集——
-- 与已结算 funding_rate 分口径；guard 前置判据数据源）。
ALTER TABLE md_funding ADD COLUMN IF NOT EXISTS next_funding_rate numeric;
