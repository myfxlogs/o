-- 0010 账本（G9 单源货币真相）：fin_cashflow+fin_ledger_snapshot+对账容差键。
-- 禁推断记账——source_ref 强制关联来源行；kind/ccy 闭集 CHECK。
CREATE TABLE IF NOT EXISTS fin_cashflow (
    id           bigserial PRIMARY KEY,
    ts           timestamptz NOT NULL DEFAULT now(),
    kind         text NOT NULL CHECK (kind IN
      ('fee','funding_settle','trade_pnl','rollback_cost','deposit','withdraw')),
    venue        text NOT NULL CHECK (venue NOT LIKE '%:%'),
    ccy          text NOT NULL CHECK (ccy IN ('USDT','USD')),
    amount       numeric NOT NULL,
    position_ref bigint,
    source_ref   bigint NOT NULL,   -- 来源行 PK（腿行/结算行/快照行——无来源禁写）
    source_table text NOT NULL,     -- 来源表名（双列定位：无 FK 跨表约束可行）
    occurred_at  timestamptz NOT NULL
);
COMMENT ON COLUMN fin_cashflow.venue IS 'venue 域（裸值禁冒号投影）';
COMMENT ON COLUMN fin_cashflow.ccy IS '本位闭集（OKX=USDT/MT5=USD，各记各的零换算；杂币不入账）';

ALTER TABLE fin_position
    ADD COLUMN IF NOT EXISTS settled_amount numeric,
    ADD COLUMN IF NOT EXISTS settled_ccy text,
    ADD COLUMN IF NOT EXISTS settled_at timestamptz;

CREATE TABLE IF NOT EXISTS fin_ledger_snapshot (
    id        bigserial PRIMARY KEY,
    venue     text NOT NULL CHECK (venue NOT LIKE '%:%'),
    ccy       text NOT NULL,
    net_usd   numeric NOT NULL,
    snap_date date NOT NULL,
    ts        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (venue, ccy, snap_date)
);
COMMENT ON COLUMN fin_ledger_snapshot.venue IS 'venue 域（裸值禁冒号投影）';

INSERT INTO app_config(key,value) VALUES
  ('ledger.recon_tol_usd','0.50')
ON CONFLICT (key) DO NOTHING;
