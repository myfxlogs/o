-- 0100 T-0312 md_regime 桥升域 SG 镜像两表+推送幂等锚（照 md_macro_event 先例）。
-- ①md_regime_fact/memo 同形镜像（MD 正本 md_1083——SG 侧零写径只加 SELECT 读面）；
-- ②md_regime_push 幂等去重面（同判词批重启/重推不重推）；
-- ③开关 notify.feishu.regime.enabled 默认开（业主钦定要收——照 fin_0019/0035 先例）。
CREATE TABLE IF NOT EXISTS md_regime_fact (
    id         bigserial PRIMARY KEY,
    dim        text NOT NULL,
    stance     text NOT NULL CHECK (stance IN ('risk_on','risk_off','neutral','uncertain')),
    confidence numeric,
    evidence   jsonb NOT NULL DEFAULT '{}',
    created_ts timestamptz NOT NULL DEFAULT now(),
    valid_until timestamptz
);
COMMENT ON TABLE md_regime_fact IS
  'regime 判词镜像（T-0312 第九桥）——分析席唯一写者在 PG-md 正本，SG 只读面';
CREATE TABLE IF NOT EXISTS md_regime_memo (
    id      bigserial PRIMARY KEY,
    ts      timestamptz NOT NULL DEFAULT now(),
    fact_id bigint,
    memo    text NOT NULL DEFAULT ''
);
COMMENT ON TABLE md_regime_memo IS
  'regime 中文备忘录镜像（T-0312）——O 群推送消费面';
CREATE TABLE IF NOT EXISTS md_regime_push (
    fact_id bigint PRIMARY KEY,
    ts      timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE md_regime_push IS
  'regime 判词推送幂等锚（T-0312）——同批重启/重拍不重推';
-- 保留账入册（state 不删行——小表）。
INSERT INTO fin_data_retention(table_name,class,note) VALUES
  ('md_regime_fact','state','第九桥镜像——regime 判词读面（T-0312）'),
  ('md_regime_memo','state','第九桥镜像——regime 备忘录读面'),
  ('md_regime_push','state','推送幂等锚不重推')
ON CONFLICT (table_name) DO NOTHING;
-- O 群判词推送开关（默认开——业主钦定）。
INSERT INTO app_config(key,value) VALUES
  ('notify.feishu.regime.enabled','true'::jsonb)
ON CONFLICT (key) DO NOTHING;
