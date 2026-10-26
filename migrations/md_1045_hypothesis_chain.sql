-- 0044 T-0106 假想官编制与假想卡链 md 链（T-0167 分链：假想卡/面差链=研究域归 PG-md；
-- ④判据 config 种在 fin_0045）：
--   ① fin_opp_hypothesis 假想卡（起草席产出——「这个面可能这样赚」待审卡，
--     非执行单；批准权=决策席；UNIQUE(venue,surface) 幂等）。
--   ② fin_surface_gap 链接通：state 扩 'hypothesized'（pending→卡引用回填）
--     + hypothesis_id 列（卡引用）。
--   ③ ev_gap 事件脊（0025 惯例——gap 行变唤醒假想官，D-0008-B 事件驱动正解）。
--   ④ 判据机读 config 种（起草面族清单+产品数下限——热启）。
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

ALTER TABLE fin_surface_gap ADD COLUMN IF NOT EXISTS hypothesis_id bigint;
ALTER TABLE fin_surface_gap DROP CONSTRAINT fin_surface_gap_state_check;
ALTER TABLE fin_surface_gap ADD CONSTRAINT fin_surface_gap_state_check
  CHECK (state IN ('pending','registered','rejected','autorejected','unclassified','hypothesized'));

CREATE OR REPLACE TRIGGER ev_gap_trg AFTER INSERT OR UPDATE OF state ON fin_surface_gap
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_gap');
