-- 0025 fin_order_leg 拒码留痕（T-0040）：Place 业务拒（sCode≠0）真拒因入列——
-- 排查面缺口补漏（现歧义吞掉真实拒因）。
ALTER TABLE fin_order_leg ADD COLUMN IF NOT EXISTS venue_err_code text;
ALTER TABLE fin_order_leg ADD COLUMN IF NOT EXISTS venue_err_msg text;
COMMENT ON COLUMN fin_order_leg.venue_err_code IS 'venue 业务拒码（okx sCode≠0；空=非拒）';
