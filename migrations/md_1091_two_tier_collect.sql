-- 1091 T-0401 两级采集制（粗扫全谱+晋级订阅）md 链正表：
--   md_tick_scan  = 粗扫层最新行表（REST tickers 60s 一拍全谱——发现面筛选
--     精度层；同键 upsert 最新赢，体积纪律钦定只存最新行禁 tick 流水）；
--   fin_sub_request = 晋级径临时订阅请求（评估臂提名候选符→collector
--     syncActiveSet UNION 臂拾取流式采——open=订 closed=退订/常驻移交）。
-- 桥表两链同形（fin_0124 镜像侧——断言⑥面）。

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
