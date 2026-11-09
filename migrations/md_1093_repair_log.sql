-- 1093 采集臂修复环动作账（业主令「RSI=自我发现然后自我修复」——
-- restart/quota_burned/restart_fail 全动作落库不随 journal 轮转；
-- SG 处置席投影/人工核证面，白名单=collecto/projector/research 三件）。

CREATE TABLE IF NOT EXISTS md_repair_log (
	id       bigserial PRIMARY KEY,
	ts       timestamptz NOT NULL DEFAULT now(),
	unit     text NOT NULL,
	feed     text NOT NULL,
	action   text NOT NULL,
	note     text NOT NULL DEFAULT ''
);
COMMENT ON TABLE md_repair_log IS
	'采集臂修复环动作账（md_1093——restart/quota_burned/restart_fail 落账，白名单三件外单元永不现于本表）';
