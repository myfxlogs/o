-- fin_0219_okxgrid.sql —— 工单#25840 OKX 原生网格适配器账本面：
-- 「bot 单」实体+周期权益快照（票面④钦定取值：venue 原生 bot 内部自管
-- 网格——仓位≠fin_position 逐腿形，记实体+快照不逐腿）。fin 域正本
-- （消费面=cmd/core okxgrid Runner——md 桥副本需求缺席不桥，待消费面
-- 出现再桥先例同族）。

-- bot 单实体账——algo_id 唯一（venue 单幂等键；strategy_id 可空=手动期
-- 业主径拉起未必挂策略链）。
CREATE TABLE IF NOT EXISTS fin_bot_order (
    id            bigserial PRIMARY KEY,
    strategy_id   bigint,
    venue         text NOT NULL DEFAULT 'okx',
    inst_id       text NOT NULL,
    algo_id       text NOT NULL UNIQUE,
    algo_ord_type text NOT NULL DEFAULT 'grid',
    params        jsonb NOT NULL,          -- {maxPx,minPx,gridNum,quoteSz,runType}
    state         text NOT NULL DEFAULT 'launched'
                  CHECK (state IN ('launched','stopping','stopped','killed')),
    launched_at   timestamptz NOT NULL DEFAULT now(),
    stopped_at    timestamptz,
    last_details  jsonb,                   -- 末拍 Details 全量原行（快照账保载荷）
    last_synced_at timestamptz
);
CREATE INDEX IF NOT EXISTS fin_bot_order_live ON fin_bot_order(state) WHERE state='launched';

-- 周期权益快照账——Details 解析列+原行双存（协议面取证：载荷全保非列裁剪）。
CREATE TABLE IF NOT EXISTS fin_bot_equity_snap (
    id            bigserial PRIMARY KEY,
    bot_id        bigint NOT NULL REFERENCES fin_bot_order(id),
    algo_id       text NOT NULL,
    inst_id       text NOT NULL,
    ts            timestamptz NOT NULL DEFAULT now(),
    investment    numeric,      -- 投入额
    grid_profit   numeric,      -- 已实现网格利（withdraw-income 收割源）
    float_profit  numeric,      -- 浮盈
    total_pnl     numeric,
    trade_num     bigint,       -- 成交数
    arbitrage_num bigint,       -- 套利成交数（网格利引擎计数）
    run_px        numeric,      -- 运行价
    fee           numeric,      -- 累计费（合约域——spot 恒空）
    fee_ccy       text,
    funding_fee   numeric,
    details       jsonb NOT NULL
);
CREATE INDEX IF NOT EXISTS fin_bot_equity_snap_bot ON fin_bot_equity_snap(bot_id, ts DESC);

-- 参数种（写域册同步：live_switch=owner 业主首闸；余=dm 管护参数）——
-- 手动期首闸钦定（票面③）：首批拉起须业主令显式放，缺键/非 true
-- fail-closed 不拉起。值形=串 "false"（GetBool 双形同族先例）。
INSERT INTO app_config(key,value) VALUES
  ('okxgrid.live_switch',    '"false"'),  -- owner 首闸：真钱拉起放行线
  ('okxgrid.snap_sec',       '300'),      -- 权益快照/监控拍秒
  ('okxgrid.max_invest_usd', '100'),      -- 单 bot 投入帽（业主闸下二次保险）
  ('okxgrid.kill_stop_type', '1'),        -- 拔线回收形：1=卖基币回quote/2=留基币
  ('okxgrid.harvest_min_usd','5')         -- 收割阈（grid_profit 达阈可 withdraw-income）
ON CONFLICT (key) DO NOTHING;

