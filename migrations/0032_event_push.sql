-- 0031 经营事件报开关（T-0061：与告警 notify.feishu.enabled 分离——业主可只关事件报；
-- 缺=false fail-closed 静默）。
INSERT INTO app_config(key,value) VALUES ('notify.feishu.events.enabled','"false"') ON CONFLICT (key) DO NOTHING;
