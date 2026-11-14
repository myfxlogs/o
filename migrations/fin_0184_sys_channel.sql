-- fin_0184 飞书双频分流（业主令 2026-11-15：两群——钱径=开平仓/日历/余额净值
-- 文本流照走 feishu.chat_id 零改；系统=运行情况/在办声明/系统告警富卡片流
-- 走新 feishu.sys_chat_id；空=fail-closed 回落主通道零漏件）。
INSERT INTO app_config(key,value) VALUES
  ('feishu.sys_chat_id','""'::jsonb),
  ('notify.feishu.sys.enabled','true'::jsonb)
ON CONFLICT (key) DO NOTHING;
