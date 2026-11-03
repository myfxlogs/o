-- 1083 T-0310 真懂行情·外部事实摄入+regime 分析席两表（md 域——阶段一）。
-- ①md_ctx_fact 原始事实 append-only（antclaw/人工/feed 三源混灌；kind 开放
--   词集不 CHECK——news|macro|oi|chain|manual 现词；人工注入径=psql insert
--   source='manual'）。
-- ②md_regime_fact 分析席结构化判词（stance 枚举域限自由发挥——注入双层防
--   第二件；决策席消费径 WHERE valid_until>now() 取活在判词）。
-- ③md_regime_memo 中文备忘录段（引证据·标数据边界）。
CREATE TABLE IF NOT EXISTS md_ctx_fact (
    id          bigserial PRIMARY KEY,
    ts          timestamptz NOT NULL DEFAULT now(),
    source      text NOT NULL,
    kind        text NOT NULL,
    payload     jsonb NOT NULL DEFAULT '{}',
    received_ts timestamptz NOT NULL DEFAULT now(),
    valid_until timestamptz
);
COMMENT ON TABLE md_ctx_fact IS
  '外部原始事实 append-only（T-0310）——不可信文本域，分析席 prompt 隔读';
CREATE INDEX IF NOT EXISTS md_ctx_fact_kind_ts ON md_ctx_fact(kind,ts DESC);

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
  'regime 判词（T-0310）——分析席唯一写者；决策席裁决读面（活在判词=valid_until>now()）';

CREATE TABLE IF NOT EXISTS md_regime_memo (
    id      bigserial PRIMARY KEY,
    ts      timestamptz NOT NULL DEFAULT now(),
    fact_id bigint,
    memo    text NOT NULL DEFAULT ''
);
COMMENT ON TABLE md_regime_memo IS
  'regime 中文备忘录（T-0310）——引证据·标数据边界；驾驶舱摘要卡二期消费';

-- 分析席写域收窄：analyst_rw 仅两表 INSERT（无 UPDATE/DELETE/fin_*——
-- 角色存在才授，测试库/fresh DB 零失败；research_rw 不经本径）。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='analyst_rw') THEN
    GRANT INSERT ON md_regime_fact, md_regime_memo TO analyst_rw;
    GRANT USAGE ON SEQUENCE md_regime_fact_id_seq, md_regime_memo_id_seq TO analyst_rw;
  END IF;
END $$;
