-- 0033 fin_position 开时基线失衡列（T-0064 漂移语义正名）：guard 首拍实算
-- 双腿名义差/均值（signed bps）落列=delta_drift 判据基线——漂移=对基线的
-- 偏移非绝对失衡（粒度残差 canon lotsz-granularity-vs-hedge-band 裁决案）。
ALTER TABLE fin_position ADD COLUMN IF NOT EXISTS open_imbalance_bps numeric;
COMMENT ON COLUMN fin_position.open_imbalance_bps IS '开时基线失衡（signed bps，guard 首拍实算——NULL=基线未立）';
