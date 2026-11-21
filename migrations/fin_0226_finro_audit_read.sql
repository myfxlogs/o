-- fin_0226: 法证读面两件（浑水式法证 2026-11-21 劈④⑤整改）。
-- ① o_fin_ro 读权开闸——钱账核表(fin_bot_order/fin_bot_equity_snap/
--   fin_paper_fill/fin_holding)连 finance 只读角色都不可读,宣告vs
--   物理对拍断径:判官快照账/权益快照对外部审计是黑盒。SELECT 授权
--   不泄写(只读角色无写权),法证对拍径恢复。
-- ② fin_cashflow_domain 域视图——业主钱(deposit/withdraw)+服务钱
--   (owner_in/out)+靓号钱(tron venue)+套利账+演习域同簿,基金口径
--   靠 kind 抽取无账户隔离(劈⑤);视图按 (venue,kind) 机读分域,
--   消费面取净口径不动物理账。
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
        EXECUTE 'GRANT SELECT ON fin_bot_order, fin_bot_equity_snap, fin_paper_fill, fin_holding TO o_fin_ro';
    END IF;
END $$;

CREATE OR REPLACE VIEW fin_cashflow_domain AS
SELECT *,
    CASE
        WHEN venue = 'sim' THEN 'sim'
        WHEN venue = 'tron' THEN 'vanity'
        WHEN kind IN ('owner_in', 'owner_out') THEN 'owner_service'
        WHEN kind IN ('deposit', 'withdraw') THEN 'arb_capital'
        ELSE 'arb_operating'
    END AS domain
FROM fin_cashflow;

COMMENT ON VIEW fin_cashflow_domain IS '现金流域分视图（fin_0226 法证劈⑤——domain: sim/vanity/owner_service/arb_capital/arb_operating;基金口径取 arb_* 两域,tron 靓号与业主服务钱物理另域)';
