-- 0018 飞书推送键（T-0028：enabled 热启+chat_id 绑定位——chat_id 业主一次性动作后填键；
-- 缺键=fail-closed 降级不推不炸）。凭据在 app_secrets 密文面非本表。
INSERT INTO app_config(key,value) VALUES
  ('notify.feishu.enabled','true'),
  ('feishu.chat_id','""')
ON CONFLICT (key) DO NOTHING;
