-- 0132 T-0428 平仓记忆补强——attempt 败因落表+trigger 判词回写。
-- 病灶（pos34 实证——attempts=5 败因零落表+trigger_kind 记谎双缺口）：
--   ① fin_guard_close 无败因列——守卫每拍平仓尝试的「为什么败」（Earn 收走
--     余额→计划即败类）只活容器日志，重建即死：教训不可复用。
--     last_attempt_at=attempt 递增同拍时点（markClosing 槽）；last_error=
--     最近一次败因（逐次覆盖为最新——带分类前缀：plan:=计划期败（余额不足/
--     参数歧义类）/venue:=venue 期败（venue 返回码），不另开列）；
--   ② fin_position.trigger_kind 记谎（delta_drift 实为单位错配幻觉）无修正
--     位——trigger_verdict=判词回写位（postmortem/裁定判词）：原 trigger_kind
--     不动（留原貌=审计链），trigger_verdict 记修正判——'confirmed'=复核
--     确认原判；异常件=机械修正码集（fast_close/settled_over/trigger_unknown/
--     bare_bag），裁定席语义修正（unit_mismatch_hallucination 类）可后覆写。
-- 三列全可空零默认——存量行零回填义务（禁回填：败因/判词只对在飞件有意义，
--   历史行无 attempt 重放源，回填即造假）。
ALTER TABLE fin_guard_close ADD COLUMN IF NOT EXISTS last_error text;
ALTER TABLE fin_guard_close ADD COLUMN IF NOT EXISTS last_attempt_at timestamptz;
ALTER TABLE fin_position ADD COLUMN IF NOT EXISTS trigger_verdict text;
COMMENT ON COLUMN fin_guard_close.last_error IS '最近一次平仓尝试败因（T-0428 逐次覆盖——前缀 plan:=计划期败/venue:=venue 期败；NULL=未败或耗尽前人工收）';
COMMENT ON COLUMN fin_guard_close.last_attempt_at IS '最近一次平仓尝试时点（T-0428 attempts 递增同拍——重试节拍审计锚）';
COMMENT ON COLUMN fin_position.trigger_verdict IS '触发判词回写位（T-0428——trigger_kind 原貌不动为审计链；confirmed=复核确认，异常=修正码集/裁定席语义修正可覆写）';
