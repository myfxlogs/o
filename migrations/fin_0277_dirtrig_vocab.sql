-- 0277 方向判据触发词入结案认知集+应急平词补册（O-41388——dir_* 判据族
-- 「码面产词未同步认知集」陈废病灶根修：dirtrig.go 四词与
-- emergency_exit 落成后 guard.postmortem.known_triggers 未补词，
-- dir_take_profit×10/dir_stop_loss×7/dir_signal_flip×1 结案全被
-- postmortem 误打 trigger_unknown 异常码烧诊断额度）。
-- ①补词五件（幂等追加——fin_0167 同径）；
-- ②历史判词净账：trigger_verdict 恰='trigger_unknown' 且 fin_guard_close
--   trigger_kind ∈ 新册词者撤码复净（trigger_kind 原貌审计链在
--   fin_guard_close 不动；多码并存行不整撤）。
UPDATE app_config SET value = value || '"dir_stop_loss"'::jsonb
WHERE key='guard.postmortem.known_triggers'
AND NOT (value @> '"dir_stop_loss"'::jsonb);
UPDATE app_config SET value = value || '"dir_take_profit"'::jsonb
WHERE key='guard.postmortem.known_triggers'
AND NOT (value @> '"dir_take_profit"'::jsonb);
UPDATE app_config SET value = value || '"dir_ttl"'::jsonb
WHERE key='guard.postmortem.known_triggers'
AND NOT (value @> '"dir_ttl"'::jsonb);
UPDATE app_config SET value = value || '"dir_signal_flip"'::jsonb
WHERE key='guard.postmortem.known_triggers'
AND NOT (value @> '"dir_signal_flip"'::jsonb);
UPDATE app_config SET value = value || '"emergency_exit"'::jsonb
WHERE key='guard.postmortem.known_triggers'
AND NOT (value @> '"emergency_exit"'::jsonb);
UPDATE fin_position p SET trigger_verdict=NULL FROM fin_guard_close g
WHERE g.position_id=p.id
  AND p.trigger_verdict='trigger_unknown'
  AND g.trigger_kind IN ('dir_stop_loss','dir_take_profit','dir_ttl',
    'dir_signal_flip','emergency_exit');
