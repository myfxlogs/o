-- 0012 持有资产台账（原币登记非运营资金——不入 cashflow/净额；两区并列展示底数）。
CREATE TABLE IF NOT EXISTS fin_holding (
    id             bigserial PRIMARY KEY,
    venue          text NOT NULL CHECK (venue NOT LIKE '%:%'),
    kind           text NOT NULL CHECK (kind IN
      ('spot_dust','earn_flexible','earn_structured','structured_order')),
    ccy            text NOT NULL,          -- 原币登记不折算（杂币存在性入账——非运营账闭集）
    product        text NOT NULL DEFAULT '',
    amount         numeric,
    notional       numeric,                -- 结构单名义额
    inherited_from text NOT NULL DEFAULT 'phase' CHECK (inherited_from IN ('nox','phase')),
    venue_ref      text NOT NULL DEFAULT '', -- 单号/产品 id
    source         text NOT NULL DEFAULT 'api' CHECK (source IN ('api','owner_declared')),
    val_ref        jsonb NOT NULL DEFAULT '{}', -- venue 报告值（非折算）
    asof_ts        timestamptz NOT NULL DEFAULT now(),
    stale          boolean NOT NULL DEFAULT false, -- 采不到=陈旧标记不抹值
    UNIQUE (venue, kind, ccy, venue_ref)
);
COMMENT ON COLUMN fin_holding.venue IS 'venue 域（裸值禁冒号投影）';
COMMENT ON COLUMN fin_holding.stale IS '陈旧标记——采集失败标陈旧不抹值';

-- NOX 遗留播种：双币结构单 call×1（12 月到期；API 枚举不到——业主口述名义额挂
-- source=owner_declared，venue_ref 待 venue 单号取证补）。
INSERT INTO fin_holding(venue,kind,ccy,product,notional,inherited_from,source,val_ref)
VALUES ('okx','structured_order','USD','dual-currency-call-dec',NULL,'nox','owner_declared',
        '{"note":"NOX 遗留双币结构单 call×1（12 月到期）；名义额待业主口述补录"}')
ON CONFLICT (venue,kind,ccy,venue_ref) DO NOTHING;
