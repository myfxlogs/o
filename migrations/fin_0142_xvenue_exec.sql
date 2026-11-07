-- 0142 T-0145 跨所 carry 真执行——双腿原子序+败一腿处置+untradeable 联动：
-- ① fin_order_leg.venue_ticket=venue 仓凭据（MT5 ticket——hedge 户
--    OrderClose 必携凭据；okx 腿=NULL 略）；
-- ② sz_unit 枚举扩 'lot'（mt5 CFD 手数量纲——下单/平仓同方言对齐）；
-- ③ exec.xvenue_open 暗发闸（默认 false——钱径新径默认关=灰度铁律，
--    venue 语义彩排过后 dmconfig 翻 true 开闸）。
ALTER TABLE fin_order_leg ADD COLUMN IF NOT EXISTS venue_ticket bigint;
COMMENT ON COLUMN fin_order_leg.venue_ticket IS
  'venue 仓凭据（T-0145）：MT5 持仓 ticket——OrderClose 必携；无票 venue=NULL';

ALTER TABLE fin_order_leg DROP CONSTRAINT IF EXISTS fin_order_leg_sz_unit_check;
ALTER TABLE fin_order_leg ADD CONSTRAINT fin_order_leg_sz_unit_check
  CHECK (sz_unit IN ('base','quote','contract','lot'));

INSERT INTO app_config(key,value) VALUES ('exec.xvenue_open','false')
  ON CONFLICT (key) DO NOTHING;
