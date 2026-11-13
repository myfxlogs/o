-- 0171 日K数据面 fin 链镜像（md_1094 同形——桥表双链归一断言钉；
-- SG collector 写库=fin 链，md 域只落 md 库会致 write_err——域=bridge 双落；
-- 工单#13065 方向族前置基建）。

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

-- 桥表授权面（T-0302 同款坑实证钉：漏授=投影拍 permission denied——
-- projector_rw 读本链行投影入 PG-md）。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'projector_rw') THEN
    GRANT SELECT ON md_klines TO projector_rw;
  END IF;
END $$;
