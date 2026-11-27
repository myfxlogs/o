-- O-41378 件2：venue 符号普查表 fin_venue_symbols——各平台 SymbolList+
-- SymbolParams 全量落库（业主令②：配对=表上纯函数 census×registry，
-- 拉数与判数分拍；params jsonb=裸返留证，配对判词可复演）。
-- map_state 机写：pending(新见)/auto_minted(dialect 落行)/flagged(败因
-- map_reason 全载明)/skip；canonical_ref=命中登记行（未落 dialect 前提件）。
-- 下架符=last_seen 止涂自然陈化（漂移探针⑤口径沿用，删符另行升件）。
CREATE TABLE IF NOT EXISTS fin_venue_symbols (
    venue         text NOT NULL,
    account       text NOT NULL,
    venue_symbol  text NOT NULL,
    base_guess    text NOT NULL DEFAULT '',
    suffix_guess  text NOT NULL DEFAULT '',
    family_guess  text NOT NULL DEFAULT 'other',
    ct_val        double precision,
    tick_sz       double precision,
    min_sz        double precision,
    lot_step      double precision,
    digits        integer,
    trade_mode    integer,
    profit_ccy    text,
    margin_ccy    text,
    spread_live   double precision,
    spec_hash     text,
    params        jsonb,
    map_state     text NOT NULL DEFAULT 'pending',
    map_reason    text,
    canonical_ref text,
    first_seen    timestamptz NOT NULL DEFAULT now(),
    last_seen     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (venue, account, venue_symbol)
);
CREATE INDEX IF NOT EXISTS fin_venue_symbols_map_idx
    ON fin_venue_symbols(venue, account, map_state);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'research_rw') THEN
        GRANT SELECT, INSERT, UPDATE ON fin_venue_symbols TO research_rw;
    END IF;
END $$;
