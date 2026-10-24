-- 0025 账户事实面（T-0041 业主令：全面了解不猜——venue 账户/乐器实勘 facts 单源层）。
-- md_venue_fact=UPSERT 幂等单源层（非 append-only 事实流——属性面低频刷新态）；
-- kind=account（户层）|instrument（乐器层）；value=jsonb（字符串/数字原生形）。
-- 消费律：动态项（TradeMode/posMode）下单时点现读，本表=决策输入层非放行证。
CREATE TABLE IF NOT EXISTS md_venue_fact (
    venue  text NOT NULL CHECK (venue NOT LIKE '%:%'),
    kind   text NOT NULL CHECK (kind IN ('account','instrument')),
    key    text NOT NULL,
    value  jsonb NOT NULL,
    ts     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (venue, kind, key)
);
COMMENT ON COLUMN md_venue_fact.venue IS 'venue 域（裸值禁冒号投影）';
COMMENT ON COLUMN md_venue_fact.kind IS 'account=户层|instrument=乐器层（闭集）';
