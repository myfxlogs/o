-- 0027 venue 账单源表（T-0044）：OKX /account/bills 落档——billId 主键幂等
-- （重启/重放不双记）；fin_cashflow.funding_settle/fee 行经此 source_ref 关联。
CREATE TABLE IF NOT EXISTS fin_venue_bill (
    id       bigserial PRIMARY KEY,
    bill_id  text NOT NULL UNIQUE,
    ts       timestamptz NOT NULL,
    inst_id  text NOT NULL,
    inst_type text NOT NULL,
    ccy      text NOT NULL,
    type     text,
    sub_type text,
    bal_chg  numeric,
    fee      numeric,
    pnl      numeric,
    sz       numeric,
    px       numeric,
    raw      jsonb NOT NULL
);
COMMENT ON COLUMN fin_venue_bill.bill_id IS 'venue 账单 ID（OKX billId——幂等键兼分页游标）';
COMMENT ON COLUMN fin_venue_bill.sub_type IS 'OKX 账单子类型（173 资金费支出/174 资金费收入——官方 docs-v5 核证）';
CREATE INDEX IF NOT EXISTS idx_venue_bill_ts ON fin_venue_bill(ts DESC);
