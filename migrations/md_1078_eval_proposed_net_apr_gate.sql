-- 1078 T-0205 重飞（md 链镜像；SG fin 侧=fin_0094 同构——桥表 DDL 双链同形，
--   同一发布 boot 内双侧齐落）：fin_opp_eval 行级不变量——proposed 行 net_apr
--   列 NOT NULL。md 镜像存量同清（proposed∧NULL → unevaluable+apr_est_missing
--   如实缺席态）后加 CHECK——projector 镜像径与回滚径「DDL 相同」前提同闸钉死。
UPDATE fin_opp_eval SET verdict='unevaluable',
  evidence=evidence || '{"reason":"apr_est_missing"}'::jsonb
WHERE verdict='proposed' AND net_apr IS NULL;
ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_proposed_net_apr;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_proposed_net_apr
  CHECK (verdict <> 'proposed' OR net_apr IS NOT NULL);
