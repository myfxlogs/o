-- 0167 方向族亏损预算熔断参数播种（工单#10413——业主令⑧ 2026-11-11：
-- 族级累计亏损=实现+浮亏合并 ≥本金总额×预算比 → 族级熔断）。
-- guard.dir_loss_budget_pct：预算比热启键（暂定 0.10=本金总额 10%，
-- 改走配置面——guard.* dm 键域，dmwrite 白名单通）。
-- guard.postmortem.known_triggers 补 dir_loss_breaker——熔断强制平
-- trigger 词入结案监察已知集（未知触发词会被 postmortem 面标异）。
INSERT INTO app_config(key,value) VALUES('guard.dir_loss_budget_pct','0.10'::jsonb)
ON CONFLICT (key) DO NOTHING;
UPDATE app_config SET value = value || '"dir_loss_breaker"'::jsonb
WHERE key='guard.postmortem.known_triggers'
AND NOT (value @> '"dir_loss_breaker"'::jsonb);
