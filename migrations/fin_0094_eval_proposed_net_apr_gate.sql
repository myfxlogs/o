-- 0094 T-0205 重飞：fin_opp_eval 行级不变量——proposed 行 net_apr 列 NOT NULL。
--   首飞闸=evidence 键面，被列洞绕行（grammar writeEval 从不写列+gapscan 显式
--   NULL 直插 proposed——无头审计二巡驳回维持，4→6 行续产实证）。本闸与不变量
--   同维：存量 proposed∧NULL 行先清（unevaluable+apr_est_missing 如实缺席态，
--   upsert 每拍重产钉痕不可持——根修前置为重飞指引钦定序），后加 CHECK——
--   新写径不可能产生该行形，违者写败=机器拒（非静默语义炸）。
UPDATE fin_opp_eval SET verdict='unevaluable',
  evidence=evidence || '{"reason":"apr_est_missing"}'::jsonb
WHERE verdict='proposed' AND net_apr IS NULL;
ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_proposed_net_apr;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_proposed_net_apr
  CHECK (verdict <> 'proposed' OR net_apr IS NOT NULL);
