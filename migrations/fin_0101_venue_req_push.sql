-- 0101 T-0312 第二刀：venue 注册请求推送幂等锚（锚键=(venue,request_at)对
-- 非 venue 单键——30d 重发闸下同 venue 二次请求 request_at 不同须各推；
-- fin 域写者=o-core 本体池 md_regime_push 同款；SG 侧 projector 零锚表写径）。
CREATE TABLE IF NOT EXISTS md_venue_req_push (
    venue      text        NOT NULL,
    request_at timestamptz NOT NULL,
    pushed_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (venue, request_at)
);
COMMENT ON TABLE md_venue_req_push IS
  'venue 注册请求推送幂等锚（T-0312）——recommended 翻牌每次请求各推一次';
INSERT INTO fin_data_retention(table_name,class,note) VALUES
  ('md_venue_req_push','state','推送幂等锚不重推（T-0312）')
ON CONFLICT (table_name) DO NOTHING;
-- O 群 venue 注册请求推送开关（默认开——业主钦定要收）。
-- 钱径通知开关：venue_req 注册请求归钱径列，键值及投递行为不变。
INSERT INTO app_config(key,value) VALUES
  ('notify.feishu.venue_req.enabled','true'::jsonb)
ON CONFLICT (key) DO NOTHING;
