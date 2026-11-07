-- 0133 T-0422 实现盈亏统一口径（三面同读——口径定义节载 docs/运维/周报口径.md）：
--   ① rollback_cost 补入 kind→class 册（income——回滚实耗是真损；册缺致
--     class='' 出 income 面=日报 net 含而 cum_nav 漏的钉洞根修）；
--   ② class 漂移行重刷（trigger 只管新写——历史漏标/改标行回填对齐册）；
--   ③ fin_report 派生行按统一口径重算（net/cum_nav 及派生 annualized/
--     net_vs_earn——读面行幂等可重算非事实改判；cum_nav 补 period_end
--     上界钉「期内累计」语义——旧无上界吞未来行同病灶回填）。
INSERT INTO fin_cashflow_kind_class(kind,class,note) VALUES
  ('rollback_cost','income','回滚成本——开/平失败反转实耗，真亏计亏（T-0422 册洞补）')
ON CONFLICT (kind) DO NOTHING;
UPDATE fin_cashflow c SET class=m.class
FROM fin_cashflow_kind_class m WHERE c.kind=m.kind AND c.class IS DISTINCT FROM m.class;

WITH calc AS (
  SELECT r.period, r.period_start,
    COALESCE((SELECT sum(amount) FROM fin_cashflow
        WHERE kind IN ('interest','funding_settle','trade_pnl','fee','rollback_cost')
          AND occurred_at>=r.period_start AND occurred_at<r.period_end),0)
    + COALESCE((SELECT sum(c.amount) FROM fin_cashflow c
        JOIN fin_position p ON p.id=c.position_ref
        WHERE c.kind IN ('spot_buy','spot_sell')
          AND p.closed_at>=r.period_start AND p.closed_at<r.period_end),0)
    + COALESCE((SELECT sum(p2.bare_bag_usd) FROM fin_position p2
        WHERE p2.closed_at>=r.period_start AND p2.closed_at<r.period_end),0) AS net,
    COALESCE((SELECT sum(amount) FROM fin_cashflow
        WHERE kind IN ('interest','funding_settle','trade_pnl','fee','rollback_cost')
          AND occurred_at<r.period_end),0)
    + COALESCE((SELECT sum(c.amount) FROM fin_cashflow c
        JOIN fin_position p ON p.id=c.position_ref
        WHERE c.kind IN ('spot_buy','spot_sell') AND p.closed_at<r.period_end),0)
    + COALESCE((SELECT sum(p3.bare_bag_usd) FROM fin_position p3
        WHERE p3.closed_at<r.period_end),0) AS cum
  FROM fin_report r
)
UPDATE fin_report r SET metrics = r.metrics
  || jsonb_build_object('net', c.net, 'cum_nav', c.cum)
  || CASE WHEN r.period IN ('weekly','monthly')
          AND (r.metrics->>'capital_base')::float8 > 0
      THEN jsonb_build_object('annualized', c.net / (r.metrics->>'capital_base')::float8
          * CASE WHEN r.period='weekly' THEN 52.0 ELSE 12.0 END)
      ELSE '{}'::jsonb END
  || CASE WHEN r.metrics ? 'net_vs_earn'
      THEN jsonb_build_object('net_vs_earn',
          c.net - ((r.metrics->>'net')::float8 - (r.metrics->>'net_vs_earn')::float8))
      ELSE '{}'::jsonb END
FROM calc c WHERE r.period=c.period AND r.period_start=c.period_start;
