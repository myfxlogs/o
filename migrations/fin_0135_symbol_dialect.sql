-- 135 T-0432 方言对表面——MT5 符号映射方言表（映射=确认件非推断件：候选
-- 机造+席位挑+机器硬验后落行，禁启发式直落）。官话唯一天：内部策略/仓位/
-- 台账/告警只用 canonical inst_id；方言只活 venue 边界——运行时查无映射=
-- fail-closed 升件 ops_event，禁静默 fallback 禁官名硬发。
-- 粒度=(venue,account)——换服务器组=重映射不继承品牌级（票面钦定）。
-- 字段=票面钦定集 {venue,account,venue_symbol,canonical,ctVal,tickSz,minSz,
-- 手数单位}+硬验在案面（digits/trade_mode/lot_step/spec_hash/quote_snapshot/
-- verified_at——漂移探针⑤基线）。venue 语义词全取自 mtapi 实返禁自造：
-- ct_val=SymbolInfo.ContractSize（1 手 base 币量）；tick_sz=TickSize>0 实值
-- 否则 Points（Exness 实拍 tick 双字段 0 不填 points 实填——T-0143 判据）；
-- min_sz/lot_step=SymGroup.MinLots/LotsStep；trade_mode=SymGroup.TradeMode。
-- quote_snapshot=硬验 GetQuote+点差快照（protojson SymbolVerifyEvidence 入
-- jsonb——架构律跨面载荷 proto 唯一源）。

CREATE TABLE IF NOT EXISTS fin_symbol_dialect (
    venue          text NOT NULL,
    account        text NOT NULL,
    canonical      text NOT NULL,
    venue_symbol   text NOT NULL,
    ct_val         double precision NOT NULL,
    tick_sz        double precision NOT NULL,
    min_sz         double precision NOT NULL,
    lot_unit       text NOT NULL,
    lot_step       double precision NOT NULL,
    digits         integer NOT NULL,
    trade_mode     integer NOT NULL,
    spec_hash      text NOT NULL,
    quote_snapshot jsonb,
    verified_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (venue, account, canonical),
    UNIQUE (venue, account, venue_symbol)
);
COMMENT ON TABLE fin_symbol_dialect IS '方言对表面（T-0432——映射=确认件：席位挑+硬验落行；运行时查无映射 fail-closed 升件禁官名硬发；粒度=(venue,account) 换组重映射不继承）';
COMMENT ON COLUMN fin_symbol_dialect.canonical IS '官话 inst_id（内部策略/仓位/台账唯一用名——OKX 形）';
COMMENT ON COLUMN fin_symbol_dialect.venue_symbol IS 'broker 方言名（venue SymbolList 自报实读——禁猜名）';
COMMENT ON COLUMN fin_symbol_dialect.ct_val IS '1 手 base 币量（venue SymbolParams.ContractSize 实返）';
COMMENT ON COLUMN fin_symbol_dialect.tick_sz IS '价格点距（TickSize>0 实值否则 Points——T-0143 venue 方言判据）';
COMMENT ON COLUMN fin_symbol_dialect.spec_hash IS 'SymbolParams 语义指纹（漂移探针⑤基线——后缀改/下架/换组/ctVal 变即升件）';
CREATE INDEX IF NOT EXISTS fin_symbol_dialect_venue ON fin_symbol_dialect(venue, account);
