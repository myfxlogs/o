-- 0127 宏观事件提醒开关键（T-0410：seed→提醒件——md_macro_event 今/明日
-- event_date 行日拍飞书推；开关独立可关照 fin_0035/0101/0119 先例）。
-- 信息面非钱径通知开关：只控制宏观事件提醒，不产生或代表资金事件。
-- 默认开——业主钦定要收（照 fin_0101/0102/0119 先例非 0035 关态）。
INSERT INTO app_config(key,value) VALUES
  ('notify.feishu.macro_remind.enabled','true'::jsonb)
ON CONFLICT (key) DO NOTHING;
