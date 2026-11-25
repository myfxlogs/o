-- fin_0264 margin-short 执行面（O-40707——39460 能力面的接线残件）。
-- ① fin_margin_loan 借币仓负债台账：借入量/息率快照/累计已记息/状态机
--    （open→closing→repaid；closing=平仓序在途——flatten 先 repay 后钉死，
--    venue 无 clOrdId 域禁盲重试由本表+历史对账担）。
-- ② fin_cashflow kind+'borrow_interest'——借币息负腿（class=income 同 fee
--    径——负额真亏计亏；息入净硬条款承续：成本必进 net 不入假正收益）。
-- ③ 闸键五枚（config-write-domain.tsv=dm 域同登）：
--    margin.collect_ivl_s   采集+息腿+健康闸拍距（缺=细胞让位不猜）
--    margin.exec_enabled    执行径总闸——前置三件（API借贷权/enableSpotBorrow/
--                           抵押物 adjEq）业主面实证全通（spec 补录 2026-11-25），
--                           仍缺省 false=接线待命态，业主/裁定开闸才武装；
--    margin.ratio_warn      负债/抵押比预警线（升件）
--    margin.ratio_kill      平仓序触发线（先 flatten 后 repay）
--    margin.max_open_usd    单仓借币名义上限
CREATE TABLE IF NOT EXISTS fin_margin_loan (
    id               bigserial PRIMARY KEY,
    venue            text        NOT NULL DEFAULT 'okx',
    ccy              text        NOT NULL,              -- 借入币（被卖出现货的基币）
    inst_id          text        NOT NULL,              -- 现货对（ccy-USDT）
    amt              double precision NOT NULL,         -- 借入量（币单位正数=负债额）
    apr              double precision NOT NULL,         -- 借入时年化息率快照
    status           text        NOT NULL DEFAULT 'open'
                     CHECK (status IN ('open','closing','repaid','flatten_failed')),
    strategy_ref     bigint,
    position_ref     bigint,                            -- fin_position.id
    execution_id     bigint,                            -- fin_execution.id
    opened_at        timestamptz NOT NULL DEFAULT now(),
    last_accrual_at  timestamptz NOT NULL DEFAULT now(),
    accrued_interest double precision NOT NULL DEFAULT 0,-- 已入账息累计（币单位）
    repaid_at        timestamptz,
    note             text        NOT NULL DEFAULT ''
);
-- 单币单在借=一仓（并发双借同币=额度/息率不可分——venue 侧亦按币聚合）。
-- mig-rehearsal-ok: 新表零存量行——唯一索引建库期不可能撞存量违例。
CREATE UNIQUE INDEX IF NOT EXISTS fin_margin_loan_one_open
    ON fin_margin_loan (venue, ccy) WHERE status IN ('open','closing');
CREATE INDEX IF NOT EXISTS fin_margin_loan_open ON fin_margin_loan (status) WHERE status IN ('open','closing');

INSERT INTO fin_surface_lifecycle(surface,accumulates,consumer_ref,terminal_path) VALUES
  ('fin_margin_loan', true, 'margin collect beat+health gate',
   'beat 拍：息腿 accrual→fin_cashflow borrow_interest；ratio≥kill/策略退役→flatten→repay→repaid')
ON CONFLICT (surface) DO NOTHING;

ALTER TABLE fin_cashflow DROP CONSTRAINT IF EXISTS fin_cashflow_kind_check;
ALTER TABLE fin_cashflow ADD CONSTRAINT fin_cashflow_kind_check
	CHECK (kind = ANY (ARRAY['fee','funding_settle','trade_pnl','rollback_cost',
		'deposit','withdraw','earn_in','earn_out','interest','recon_adjust',
		'spot_buy','spot_sell','owner_in','owner_out',
		'dust_in','dust_out','supply_income','airdrop','unmapped_bill',
		'dcd_out','dcd_in','borrow_interest'])) NOT VALID;
ALTER TABLE fin_cashflow VALIDATE CONSTRAINT fin_cashflow_kind_check;

INSERT INTO fin_cashflow_kind_class(kind,class,note) VALUES
	('borrow_interest','income','保证金借币息负扣——真亏计亏（同 fee 径负额入账，O-40707）')
ON CONFLICT (kind) DO NOTHING;

INSERT INTO app_config(key,value) VALUES
  ('margin.collect_ivl_s','300'),
  ('margin.exec_enabled','false'),
  ('margin.ratio_warn','0.60'),
  ('margin.ratio_kill','0.80'),
  ('margin.max_open_usd','500')
ON CONFLICT (key) DO NOTHING;

-- 升件惧值册（ops_esc_pending 消费源——kill=近平仓动作高惧、warn=预警档）。
INSERT INTO ops_esc_valence(esc_key,weight,note) VALUES
  ('margin_ratio_warn',55,'margin-short 负债/抵押比预警档——逼近 kill 平仓序线（工单O-40707）'),
  ('margin_ratio_kill',80,'margin-short 负债/抵押比 kill 档——flatten→repay 平仓序已触发（工单O-40707）')
ON CONFLICT (esc_key) DO NOTHING;
