-- 1094 日K数据面（工单#13065——方向族前置基建：全库无 candle/kline 面，
-- information_schema 零命中实证）。WS candle1D 活跃集尾流+收线确认态 upsert，
-- bar 字段留多周期扩展（1D 首发）。

CREATE TABLE IF NOT EXISTS md_klines (
	venue    text        NOT NULL DEFAULT 'okx',
	inst_id  text        NOT NULL,
	bar      text        NOT NULL,
	ts       timestamptz NOT NULL,              -- K线开盘时刻（OKX 帧首元毫秒）
	o        float8      NOT NULL,
	h        float8      NOT NULL,
	l        float8      NOT NULL,
	c        float8      NOT NULL,
	vol      float8      NOT NULL,
	vol_ccy  float8      NOT NULL,
	confirm  boolean     NOT NULL DEFAULT false, -- false=未收线在形成（upsert 覆写至收线）
	recv_ts  timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (venue, inst_id, bar, ts)
);
COMMENT ON TABLE md_klines IS
	'日K数据面（md_1094 工单#13065——candle1D WS 活跃集尾流；confirm=false 为形成中帧随 tick 覆写，收线帧 confirm=true 必落）';
