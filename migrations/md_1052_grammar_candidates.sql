-- 0052 模板文法化 md 链（T-0120/D-0016 Ⅱ；T-0167 分链：fin_opp_template 文法列
-- =研究域归 PG-md；fin_opp_eval direction 扩=桥表 DDL 与 fin 链 fin_0052 同形；
-- 判据键在 fin_0052）：fin_opp_template +state 列——enabled=现役执行态（存量行
-- 默认）/candidate=文法枚举候选（无 executor 不落执行径）/retired。

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
