-- 0087 maker 径腿单型别（T-0238——post_only/market 降级双径审计锚：
-- 每腿记实际执行径，费率分制对账件）。
ALTER TABLE fin_order_leg ADD COLUMN IF NOT EXISTS ord_type text NOT NULL DEFAULT 'market';
