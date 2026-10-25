-- 0046 T-0107 腿量纲混记绞肉环根修（P0 钱径）：
--   fin_order_leg.sz 双语义（ETH 币量记 vs BTC quote 额记）→ guard.notionalNow
--   按 sz×px 读 quote 额腿=$1.94M 假失衡 ~20000bps ≫500 兜底 → 开盘 140ms 秒杀
--   ×9 轮连绞（$0.05/轮）。消歧：sz_unit 语义位+qty_base=venue 实填币量
--   （accFillSz 唯一真源——请求量禁当成交记）。
--   历史行判读规则（明写）：SWAP=contract 恒真；spot 腿自证=腿级现金流毛额
--   对拍（|amount|≈sz→quote〔T-0064 quote_ccy 委托〕、|amount|≈sz×px→base）；
--   两判皆不中/px 缺/无 cashflow=sz_unit 留 NULL=未消歧——消费面 fail-closed
--   跳判（宁可漏判不误判，禁猜）。
ALTER TABLE fin_order_leg ADD COLUMN IF NOT EXISTS sz_unit text
  CHECK (sz_unit IN ('base','quote','contract'));
ALTER TABLE fin_order_leg ADD COLUMN IF NOT EXISTS qty_base numeric;
COMMENT ON COLUMN fin_order_leg.sz_unit IS
  'sz 量纲语义位（T-0107）：base=币量/quote=USDT 额/contract=张数；NULL=历史未消歧——消费面 fail-closed';
COMMENT ON COLUMN fin_order_leg.qty_base IS
  'venue 实填币量（accFillSz）——spot 腿盯市唯一真源；衍生品=NULL（张数在 sz）';

UPDATE fin_order_leg SET sz_unit='contract' WHERE inst_id LIKE '%-SWAP' AND sz_unit IS NULL;

UPDATE fin_order_leg l SET sz_unit='base', qty_base=l.sz
FROM fin_cashflow c
WHERE c.source_table='fin_order_leg' AND c.source_ref=l.id
  AND l.state='filled' AND l.inst_id NOT LIKE '%-SWAP' AND l.sz_unit IS NULL
  AND l.px IS NOT NULL AND l.sz > 0
  AND abs(abs(c.amount) - l.sz*l.px) <= 0.02*abs(l.sz*l.px)+1e-9;

UPDATE fin_order_leg l SET sz_unit='quote',
  qty_base = CASE WHEN l.px > 0 THEN abs(c.amount)/l.px END
FROM fin_cashflow c
WHERE c.source_table='fin_order_leg' AND c.source_ref=l.id
  AND l.state='filled' AND l.inst_id NOT LIKE '%-SWAP' AND l.sz_unit IS NULL
  AND l.sz > 0
  AND abs(abs(c.amount) - l.sz) <= 0.02*abs(l.sz)+1e-9;

-- 再开冷却（T-0107：guard 触发平仓后同策略同标的间隔窗——45s 级连绞禁再演；
-- 300s=5×lifecycle 节拍，热启可调）。
INSERT INTO app_config(key,value) VALUES
  ('exec.reopen_cooldown_sec', '300')
ON CONFLICT (key) DO NOTHING;
