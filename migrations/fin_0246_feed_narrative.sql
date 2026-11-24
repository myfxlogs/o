-- fin_0246 feed 叙事模式开关（工单O-36963——业主令「O·动要故事线非流水账」：
-- unit= 锚会话窗（在办→会话收口/唤起果/唤起败）聚合渲成一段白话叙事，
-- 聚合失败/无 unit 锚退回现行逐行渲染——叙事是加成非替换）：
--   app_config feed.narrative_mode=true——呈现层开关，默认开（业主是消费者，
--   他要的就是默认态）；置 false=回逐行模式。键属 owner 写域（呈现面非钱径，
--   同 notify.* 族——DM 写径不可达）。

INSERT INTO app_config(key,value,updated_by) VALUES
  ('feed.narrative_mode', 'true'::jsonb, 'fin_0246')
ON CONFLICT (key) DO NOTHING;
