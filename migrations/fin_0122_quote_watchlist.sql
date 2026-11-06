-- 0122 T-0391 自选报价条 watchlist 表（业主令 2026-11-06——「订阅某1个
-- symbol 的实时报价，这个symbol需要我来定义，要支持搜索；NOX归档总览页
-- 就有，我要的是这个效果，前端要适合现有前端」）：NOX QuoteStrip/
-- WatchlistModal 语义移植——成员 symbol 集+展示序（ord=保存时全量替换
-- 写序，零成员=如实缺态）；候选域=md_ticks DISTINCT inst_id（有行情
-- 才可选，不开任意 symbol 输入）。fin 域正本（同 fin_price_alert 域
-- 钉——报价/提醒皆 fin 侧观测面）。纯展示面：不喂判定/策略/采集域零动。
CREATE TABLE IF NOT EXISTS fin_quote_watchlist (
    symbol     text PRIMARY KEY,
    ord        int NOT NULL,
    created_by text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE fin_quote_watchlist IS '自选报价条成员表（T-0391——NOX QuoteStrip 语义移植；ord=全量替换写序，纯展示面不喂判定）';
