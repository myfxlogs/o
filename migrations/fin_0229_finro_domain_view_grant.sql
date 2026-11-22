-- fin_0229: fin_cashflow_domain 视图 o_fin_ro SELECT 授权补齐（工单O-28958）。
-- fin_0226 劈⑤发底表 4 件（fin_bot_order/fin_bot_equity_snap/
--   fin_paper_fill/fin_holding）漏授视图本体——劈⑤对拍面自身黑盒：
--   has_table_privilege 实证底表=true 而 domain_view=false。
-- 单件单令：只授本视图本体，不动其他视图/其他 role。
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
        EXECUTE 'GRANT SELECT ON fin_cashflow_domain TO o_fin_ro';
    END IF;
END $$;
