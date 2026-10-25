-- 0053 事实读面新鲜度判据键（T-0128——guard 判据链陈行=缺 fail-closed
-- 语义可调；缺省值与代码内底同锚——键缺席不废判据）。
INSERT INTO app_config(key,value) VALUES
  ('exec.tick_max_age_s',      '120'),
  ('guard.funding_max_age_min','540')
ON CONFLICT (key) DO NOTHING;
