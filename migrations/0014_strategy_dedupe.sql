-- 0014 paper 同标的去重：direction 入策略行+active 部分唯一索引
-- （同 venue+inst_id+direction 同时最多一个非 retired 策略行——armed 复用不叠仓）。
ALTER TABLE fin_strategy ADD COLUMN IF NOT EXISTS direction text NOT NULL DEFAULT 'short_perp_long_spot';
CREATE UNIQUE INDEX IF NOT EXISTS fin_strategy_active_dedupe
    ON fin_strategy(venue, inst_id, direction) WHERE state <> 'retired';
