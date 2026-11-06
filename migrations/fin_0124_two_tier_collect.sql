-- 0124 T-0401 两级采集制 fin 链镜像侧（md_1091 同形——桥表投影终表；
-- 语句文本照 md_1091 逐字，GRANT 补 T-0302 实证坑同款：桥表漏授=投影拍
-- permission denied）。projector_rw=投影写者授权面。

CREATE TABLE IF NOT EXISTS md_tick_scan (
    ts       timestamptz NOT NULL,
    venue    text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    inst_id  text NOT NULL,
    last     numeric,
    bid      numeric,
    ask      numeric,
    recv_ts  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (venue, inst_id)
);
COMMENT ON COLUMN md_tick_scan.venue IS 'venue 域（裸值禁冒号投影）';
COMMENT ON TABLE md_tick_scan IS
  '粗扫层最新行表（T-0401）——REST tickers 60s 全谱 last/bid/ask；筛选精度层，执行精度仍归 md_ticks';

CREATE TABLE IF NOT EXISTS fin_sub_request (
    ts        timestamptz NOT NULL DEFAULT now(),
    venue     text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    inst_id   text NOT NULL,
    requester text NOT NULL,
    state     text NOT NULL DEFAULT 'open' CHECK (state IN ('open','closed')),
    reason    text NOT NULL DEFAULT '',
    recv_ts   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (venue, inst_id, requester)
);
COMMENT ON COLUMN fin_sub_request.venue IS 'venue 域（裸值禁冒号投影）';
COMMENT ON TABLE fin_sub_request IS
  '晋级径临时订阅请求（T-0401）——评估臂提名非活跃符；open=activeSet UNION 臂拾取，closed=退订/提案常驻移交';

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'projector_rw') THEN
    GRANT SELECT,INSERT,UPDATE,DELETE ON md_tick_scan TO projector_rw;
    GRANT SELECT,INSERT,UPDATE,DELETE ON fin_sub_request TO projector_rw;
  END IF;
END $$;
