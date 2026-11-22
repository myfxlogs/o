-- 1101 OKX 期权链采集面（工单#32540——covered-call 研究 #32274 数据前置：
-- venue 不补历史，延一天缺一天；频道普查实证 opt-summary WS 增量推
-- delta/gamma/theta/vega/markVol 全链希腊字，tickers/books5/mark-price
-- 对 OPTION instId 全受理）。四族=BTC/ETH×{USD 反向,USD_UM 线性}。
-- 存量纪律=纯加法：新表域零碰既有频道语义/既存表（ban 钉）。
-- PG 体积纪律：md_opt_summary/md_opt_ticker=状态表 upsert 有界
--（行数=live 合约数 ~3.5k）；md_opt_snapshot=周期快照追加表，
-- 粗粒度起 optSnapEvery=60min×活跃子集（近月±20%行权价窗）——
-- 保留策：90d 证据窗，超限先降频非删表（容量器官 pg-capacity 域）。

-- ①合约目录枚举快照（instruments REST 实读全量 upsert——state 列载
-- live/preopen/suspend/delisted 全态，活链判据只取 live）。
CREATE TABLE IF NOT EXISTS md_opt_instruments (
	venue       text        NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
	inst_id     text        NOT NULL,
	inst_family text        NOT NULL,
	uly         text        NOT NULL,
	opt_type    text        NOT NULL,               -- C=看涨 P=看跌
	exp_time    timestamptz,
	strike      numeric,
	ct_type     text,                               -- inverse=币本位反向 / linear=线性
	settle_ccy  text,
	state       text        NOT NULL,
	tick_sz     numeric,
	lot_sz      numeric,
	min_sz      numeric,
	ct_val      numeric,
	ct_val_ccy  text,
	list_time   timestamptz,
	recv_ts     timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (venue, inst_id)
);
COMMENT ON TABLE md_opt_instruments IS
	'OKX 期权合约目录枚举快照（md_1101 工单#32540——instruments?instType=OPTION&instFamily 实读全态 upsert；活链判据=state=live）';

-- ②期权摘要最新态（opt-summary WS 增量推逐行 upsert——delta/gamma/
-- theta/vega/markVol=IV/bidVol/askVol/volLv/fwdPx/distance；
-- 空串字段落 NULL 不落 0——缺席≠真值0 标缺律）。
CREATE TABLE IF NOT EXISTS md_opt_summary (
	venue     text        NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
	inst_id   text        NOT NULL,
	ts        timestamptz NOT NULL,               -- venue 帧 ts（毫秒 epoch）
	delta     numeric,
	gamma     numeric,
	theta     numeric,
	vega      numeric,
	mark_vol  numeric,                            -- 期权 IV（markVol 字段）
	bid_vol   numeric,
	ask_vol   numeric,
	vol_lv    numeric,
	fwd_px    numeric,
	distance  numeric,                            -- 行权价距现价比（venue 供）
	buy_apr   numeric,
	sell_apr  numeric,
	lever     numeric,
	recv_ts   timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (venue, inst_id)
);
COMMENT ON TABLE md_opt_summary IS
	'OKX 期权摘要最新态（md_1101 工单#32540——opt-summary WS instFamily 增量推 upsert；per-inst 写节流 60s 防增量洪）';

-- ③活跃链 ticker 最新态（tickers WS 订阅子集——近 N 到期日×现价±20%
-- 行权价窗；last 可空=无成交，bid/ask 空串落 NULL）。
CREATE TABLE IF NOT EXISTS md_opt_ticker (
	venue    text        NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
	inst_id  text        NOT NULL,
	ts       timestamptz NOT NULL,
	last     numeric,
	last_sz  numeric,
	bid_px   numeric,
	bid_sz   numeric,
	ask_px   numeric,
	ask_sz   numeric,
	vol_24h  numeric,
	recv_ts  timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (venue, inst_id)
);
COMMENT ON TABLE md_opt_ticker IS
	'OKX 期权活跃链 ticker 最新态（md_1101 工单#32540——tickers WS 活跃子集 upsert，per-inst 3s 节流同 md_ticks 律）';

-- ④周期快照追加表（optSnapEvery=60min 对活跃子集全量落行——
-- covered-call IV/希腊字时序证据窗；src_ts=源行 venue ts 供新鲜度判）。
CREATE TABLE IF NOT EXISTS md_opt_snapshot (
	snapshot_ts timestamptz NOT NULL,             -- 本地拍点（每轮同值）
	venue       text        NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
	inst_id     text        NOT NULL,
	src_ts      timestamptz,                      -- md_opt_summary 源行 venue ts
	delta       numeric,
	gamma       numeric,
	theta       numeric,
	vega        numeric,
	mark_vol    numeric,
	bid_vol     numeric,
	ask_vol     numeric,
	vol_lv      numeric,
	fwd_px      numeric,
	distance    numeric,
	buy_apr     numeric,
	sell_apr    numeric,
	lever       numeric,
	recv_ts     timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (snapshot_ts, venue, inst_id)
);
COMMENT ON TABLE md_opt_snapshot IS
	'OKX 期权活跃链周期快照（md_1101 工单#32540——60min×活跃子集追加；保留策 90d 窗，超限先降频非删表）';
