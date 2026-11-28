-- md_1113 fin_valuation_ccy PG-md 只读副本（O-41395 件②——bridge-reverse
-- 镜像，fin_0279 正本对偶：SG o-core 发布器→projector 全表 diff+出集
-- DELETE 同步；collector activeSet/localWritePred 经 legset.
-- ValuationCcyArm 消费）。DDL 与 fin 链正表归一化同形（check_migrations
-- 断言⑥ 桥/反向桥同形钉）；触发器/回填不在副本。
CREATE TABLE IF NOT EXISTS fin_valuation_ccy (
    ccy        text PRIMARY KEY,
    src        text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE fin_valuation_ccy IS
  'fin_valuation_ccy PG-md 只读副本（O-41395 反向桥）——SG 正本 projector 全表 diff 维护，collector 订阅集/写谓词读';
