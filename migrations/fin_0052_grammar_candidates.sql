-- 0052 模板文法化 fin 链（T-0120/D-0016 Ⅱ；T-0167 分链）：fin_opp_eval direction
-- 扩=桥表 DDL 与 md 链 md_0052 同形（DDL 归一化断言钉）；fin_opp_template 文法列
-- 正本在 PG-md 不落 fin 链；判据键=app_config 归 PG-fin。

ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_direction_check;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell'));

INSERT INTO app_config(key,value) VALUES
  ('research.grammar.scan_interval_min', '120')
ON CONFLICT (key) DO NOTHING;
