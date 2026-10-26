-- 0050 T-0124 提案晋升模板门闸（SG 实证 748 条 DCD 期权提案被 lifecycle 建成
-- funding-hedge paper——晋升消费器不滤模板，量纲全错簿册污染）：
--   去重键绑模板——(venue,inst_id,direction,template_ref) 跨族同键不串仓。
DROP INDEX IF EXISTS fin_strategy_active_dedupe;
CREATE UNIQUE INDEX fin_strategy_active_dedupe
    ON fin_strategy(venue, inst_id, direction, template_ref) WHERE state <> 'retired';
