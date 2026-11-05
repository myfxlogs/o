-- 0114 T-0356 vol-sell 复役闸：SG fin 正本键先落 false，部署新守闸码后
-- 再由运维按票面步序翻 true；连败超过 N 用持久条件请求停扫，不越权改键。
INSERT INTO app_config(key,value) VALUES
  ('research.volsell.enabled','false'::jsonb),
  ('research.volsell.fail_streak_n','3'::jsonb)
ON CONFLICT (key) DO NOTHING;

INSERT INTO ops_event(seat,event,ticket,note) VALUES
  ('施工席','启用','T-0356',
   'vol-sell 复役配置种于 SG fin 正本：enabled=false（新 binary 生效后按步序启用）、fail_streak_n=3（严格>N持久停扫+升件）；research_rw 无 app_config 写权，异常不自动翻键');
