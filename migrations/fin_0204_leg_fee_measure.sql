-- fin_0204_leg_fee_measure.sql —— 工单#21129 摩擦费测量根修件①：
--
--   fin_order_leg 加 fee/fee_ccy 测量列——venue 返费原始值存腿。
--   测量盲三洞实证（分子=cashflow kind='fee' JOIN position_ref）：
--   ①legCash 现金账只入 feeCcy='USDT'——现货买腿费在 base 币不入账
--     （现金守恒口径正确），测量面随丢 27 张 bill≈$0.57；
--   ②ingest 杂币 bill 不入账+SPOT/MARGIN 费挂账跳过（以为 legCash
--     已盖现货费=覆盖洞）；③post() 时窗归仓 miss→position_ref NULL
--     的 fee 行被 JOIN 排掉（≈$0.80）。记 $0.3244 vs 实 ~$1.69。
--   腿级存费=按仓归集精确口径（position↔execution_id 1:1 锚），
--   friction 校准读侧非 USDT 费按腿 px 折 USDT——bill 分腿对拍。
--
--   回填：fin_venue_bill inst+px 相等+|Δts|<15s 双向最近配（每腿取
--   最近 bill、每 bill 只供一腿——部分成交拆腿同 bill 场景禁倍计；
--   同距取小 id 定序）。无配 bill=非腿径账（PENGU 手工面等 $1.93
--   未配实证）本不入仓级测量。配 96 spot+50 swap 腿/35 全配仓实证。

BEGIN;

ALTER TABLE fin_order_leg ADD COLUMN IF NOT EXISTS fee numeric;
ALTER TABLE fin_order_leg ADD COLUMN IF NOT EXISTS fee_ccy text;

WITH m AS (
  SELECT l.id AS leg_id, b.id AS bill_id, b.fee, b.ccy,
         row_number() OVER (PARTITION BY l.id
             ORDER BY abs(extract(epoch from b.ts - l.ts)), b.id) AS lr,
         row_number() OVER (PARTITION BY b.id
             ORDER BY abs(extract(epoch from b.ts - l.ts)), l.id) AS br
  FROM fin_order_leg l
  JOIN fin_venue_bill b
    ON b.inst_id = l.inst_id AND b.px = l.px AND b.fee <> 0
   AND abs(extract(epoch from b.ts - l.ts)) < 15
  WHERE l.state = 'filled'
)
UPDATE fin_order_leg l SET fee = m.fee, fee_ccy = m.ccy
FROM m WHERE l.id = m.leg_id AND m.lr = 1 AND m.br = 1;

COMMIT;
