-- 0052 模板文法化（T-0120/D-0016 Ⅱ）：fin_opp_template +state 列——
-- enabled=现役执行态（存量行默认）/candidate=文法枚举候选（无 executor
-- 不落执行径）/retired。结构键 UNIQUE(template,venue,surface) 幂等锚；
-- fin_opp_eval direction 扩三族向（basis/lending/earn 利差+vol-sell）。

ALTER TABLE fin_opp_template ADD COLUMN IF NOT EXISTS state text NOT NULL DEFAULT 'enabled';
ALTER TABLE fin_opp_template ADD CONSTRAINT fin_opp_template_state_check
  CHECK (state IN ('enabled','candidate','retired'));
CREATE UNIQUE INDEX IF NOT EXISTS uq_opp_template_struct
  ON fin_opp_template(template, venue, COALESCE(surface,''));

ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_direction_check;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell'));

-- 文法枚举节拍判据键（scan_interval_min 热启）。
INSERT INTO app_config(key,value) VALUES
  ('research.grammar.scan_interval_min', '120')
ON CONFLICT (key) DO NOTHING;
