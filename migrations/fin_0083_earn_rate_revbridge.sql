-- 0083 md_earn_rate 反向桥副本表（T-0214：bridge-reverse 域首例 md_* 表——
-- SG 正本（earn 臂写）→PG-md 只读副本，evalCarry 读面供血；DDL 与 md_1049
-- 状态表形归一（PK(venue,ccy) 最新赢 upsert；本链无 timescale——无 hypertable）。
CREATE TABLE IF NOT EXISTS md_earn_rate (
    ts    timestamptz NOT NULL DEFAULT now(),
    venue text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    ccy   text NOT NULL,
    rate  numeric NOT NULL,
    PRIMARY KEY (venue, ccy)
);
COMMENT ON COLUMN md_earn_rate.venue IS 'venue 域（裸值禁冒号投影）';
