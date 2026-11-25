-- fin_0260_manual_flow.sql —— 工单O-33207 手动资金流/手动仓采集机制：
--   「业主在 venue 侧动手，O 必须看见」——MT5 OrderHistory 交易行
--   （DealBuy/DealSell 平仓仓）产三簿记件：
--   ① fin_venue_position 手动仓事实面——非 O 单（comment≠clOrdID）
--     在仓/平仓事实逐户落行，mtm 盯市偏移并入对账净值（在仓浮盈
--     否则 equity 含它 net 不含=幻影残差）；O 自执行仓走
--     fin_order_leg/fin_position+mtm 既有径不重复入册（双簿防线）。
--   ② 平仓 realized（Profit+Swap+Commission+Fee 行级聚合）→
--     fin_cashflow kind='trade_pnl' source_table='mt5_deal'
--     source_ref=position ticket（票号命名空间与 balance deal 同
--     序列不撞——mt5_deal_uk(account_id,source_table,source_ref)
--     幂等重放零双簿）；O 单同径入账（守恒优先——venue balance
--     不分谁的手），position_ref 经 comment→leg→position 归仓，
--     手动仓 position_ref=NULL 聚合层豁免策略业绩。
--   ③ balance 邻类族扩账——Credit/Bonus 入 deposit|withdraw
--     （spec 钦定族），Charge/Correction/Commission*/InterestRate
--     落 unmapped_bill 守恒兜底（不静默吞差——金额带号入账+
--     判词注记并存，归因待裁不丢）。
--   游标=acctRun.balWM 逐户独立（venue→run 一户一件，单户读败
--   不拖他户；重启 7d 回看+幂等键兜底——account_id 粒度天然隔离，
--   禁 venue 共游标在案）。
--   fin_surface_lifecycle 登记——积面册（accumulates=true：仓位
--   行随开仓累积，终态=OrderHistory 平仓行→closed/stale 清出）。

CREATE TABLE IF NOT EXISTS fin_venue_position (
    id            bigserial PRIMARY KEY,
    account_id    text NOT NULL REFERENCES fin_account(account_id),
    venue         text NOT NULL,             -- roster venue token（数据面键）
    ticket        text NOT NULL,             -- venue position ticket（OpenedOrders/OrderHistory posTicket 同域）
    symbol        text NOT NULL,             -- venue symbol（=md_ticks/klines inst_id 同键）
    side          text NOT NULL CHECK (side IN ('buy','sell')),
    lots          numeric NOT NULL,
    contract_size numeric NOT NULL DEFAULT 0, -- venue ContractSize（利润换算乘子）
    open_px       numeric,
    open_ts       timestamptz,
    ccy           text NOT NULL DEFAULT 'USD', -- 利润币（CFD 户=账户本位；非 USD 户前端改列不改码）
    state         text NOT NULL DEFAULT 'open' CHECK (state IN ('open','closed','stale')),
    close_px      numeric,
    close_ts      timestamptz,
    realized      numeric,                   -- 平仓实收簿记额（OrderHistory 聚合）
    mtm_off       numeric,                   -- 直给盯市偏移（OKX upl 现成值径——非空则 mtm 直加不算 mark；MT5 手动仓 NULL=mark 价差径）
    last_seen     timestamptz NOT NULL DEFAULT now(), -- OpenedOrders 快照末见时刻（在仓权威=快拍）
    created_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, ticket)
);
COMMENT ON TABLE fin_venue_position IS 'venue 侧非 O 手动仓事实面（O-33207——业主手动仓入账但不计策略业绩；mtm 盯市并入对账净值；O 自执行仓禁入册=双簿防线）';
COMMENT ON COLUMN fin_venue_position.state IS 'open=在仓快照在册/closed=OrderHistory 平仓实证/stale=OpenedOrders 快照消失未获平仓实证（mtm 只计 open）';

CREATE INDEX IF NOT EXISTS fin_venue_position_open_idx
    ON fin_venue_position (account_id) WHERE state='open';

INSERT INTO fin_surface_lifecycle(surface,accumulates,consumer_ref,terminal_path) VALUES
  ('fin_venue_position', true, 'mt5 collector+ledger mtm/recon',
   'OrderHistory 平仓行→state=closed+trade_pnl 入账；OpenedOrders 快照消失→stale 清出 mtm')
ON CONFLICT (surface) DO NOTHING;
