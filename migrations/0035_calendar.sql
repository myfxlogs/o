-- 0035 财经日历事件报（T-0070：[财经日历] 前缀与 [经营事件]/告警分区；
-- md_calendar_push=幂等去重面（同事件重启/重拉不重推）；开关独立可关）。
CREATE TABLE IF NOT EXISTS md_calendar_push (
    event_key  text PRIMARY KEY,  -- title|date|time 组合键（同事件幂等锚）
    sent_at    timestamptz NOT NULL DEFAULT now()
);
INSERT INTO app_config(key,value) VALUES
  ('notify.feishu.calendar.enabled','"false"'),
  ('calendar.min_impact','"High"'),
  ('calendar.lead_minutes','"60"')
ON CONFLICT (key) DO NOTHING;
