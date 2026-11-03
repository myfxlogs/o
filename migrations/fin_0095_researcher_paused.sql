-- 0095 T-0299 研究员席暂停键（D-0043 收敛——第一分钱前只修钱径）。
-- paused=true=预期静默：silent 看门不发 research_arm_silent，改日限频
-- info note；键缺=非 paused（fail-open）。恢复=翻键，tools/hypo 与
-- fin_opp_hypothesis 不删。
INSERT INTO app_config(key,value) VALUES
  ('research.researcher.paused','true')
ON CONFLICT (key) DO NOTHING;
