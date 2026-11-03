-- 0099 T-0303 两幽灵表升桥 fin 链侧（SG 化石表→投影终表——业主面板假警
-- 实证：hypothesis SG 残 proposed|8 vs 正本 parked×10；template enabled 1 vs 3）。
-- DDL 语句集与 md 链同形（断言⑥钉——CREATE TABLE IF NOT EXISTS 对齐既有
-- SG 幽灵表，语句文本照 md_1006/1044/1045/1052/1079 逐字）。GRANT 补 T-0302
-- 实证坑（桥表漏授=projector 每拍 permission denied）。
CREATE TABLE IF NOT EXISTS fin_opp_template (
    id       serial PRIMARY KEY,
    template text NOT NULL,
    venue    text NOT NULL CHECK (venue NOT LIKE '%:%'),
    params   jsonb NOT NULL DEFAULT '{}',
    enabled  boolean NOT NULL DEFAULT true
);
COMMENT ON COLUMN fin_opp_template.venue IS 'venue 域（裸值禁冒号投影）';
COMMENT ON COLUMN fin_opp_template.template IS '机会模板型（机会模板书六型——funding 先落）';
ALTER TABLE fin_opp_template ADD COLUMN IF NOT EXISTS surface text;
ALTER TABLE fin_opp_template ADD COLUMN IF NOT EXISTS state text NOT NULL DEFAULT 'enabled';
ALTER TABLE fin_opp_template DROP CONSTRAINT IF EXISTS fin_opp_template_state_check;
ALTER TABLE fin_opp_template ADD CONSTRAINT fin_opp_template_state_check
  CHECK (state IN ('enabled','candidate','retired'));
CREATE UNIQUE INDEX IF NOT EXISTS uq_opp_template_struct
  ON fin_opp_template(template, venue, COALESCE(surface,''));

CREATE TABLE IF NOT EXISTS fin_opp_hypothesis (
    id             bigserial PRIMARY KEY,
    ts             timestamptz NOT NULL DEFAULT now(),
    venue          text NOT NULL CHECK (venue NOT LIKE '%:%'),
    surface        text NOT NULL,
    family         text NOT NULL,
    leg_primitives jsonb NOT NULL DEFAULT '[]',
    requirements   jsonb NOT NULL DEFAULT '{}',
    state          text NOT NULL DEFAULT 'proposed'
                   CHECK (state IN ('proposed','adopted','dismissed')),
    evidence       jsonb NOT NULL DEFAULT '{}',
    UNIQUE (venue, surface)
);
COMMENT ON TABLE fin_opp_hypothesis IS
  '假想卡（T-0106）——假想官起草的模板假设待审卡；批准权=决策席，卡非执行单';
ALTER TABLE fin_opp_hypothesis DROP CONSTRAINT fin_opp_hypothesis_state_check;
ALTER TABLE fin_opp_hypothesis ADD CONSTRAINT fin_opp_hypothesis_state_check
  CHECK (state IN ('proposed','adopted','dismissed','parked'));

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'projector_rw') THEN
    GRANT SELECT,INSERT,UPDATE,DELETE ON fin_opp_template TO projector_rw;
    GRANT SELECT,INSERT,UPDATE,DELETE ON fin_opp_hypothesis TO projector_rw;
  END IF;
END $$;
