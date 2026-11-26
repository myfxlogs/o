-- fin_0266 O-40829 停泊环意向豁免阈件（P0 钱径正确性缺陷——「有用途的钱」
-- 不可扫：赎回款/负债担保存钱/在途单预备金与天然闲钱同待遇被扫走实证在案）。
-- 两键缺省 true=豁免判据全活；置 false=对应判据源降级/总闸回滚旧行为。
INSERT INTO app_config(key,value) VALUES
  ('earn.sweep_intent_exempt', 'true'),  -- 意向豁免总闸（false=旧行为全复回滚径）
  ('earn.sweep_liab_check',   'true')    -- venue 负债查径开关（false=降级跳 venue 源，余判据照走）
ON CONFLICT (key) DO NOTHING;
