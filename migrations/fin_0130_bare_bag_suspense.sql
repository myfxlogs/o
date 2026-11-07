-- 0130 T-0419 平仓残袋挂账列（裸袋显形+settled 语义根修——钱径实证 2026-11-07：
--   仓#33 开腿 sz=24.999 sz_unit=quote→qty_base=1915.63 CORE，平腿把同一
--   24.99 美元额当 base 卖（仅清 1.3%），仓标 closed 记 settled=-24.67
--   假亏，1890 CORE 残袋留 venue 被 Earn 自动收息——「closed 了但货还在」
--   静默背离。
--   ① bare_bag_* 三列=未回资产挂账（ccy/币量/挂账时估值 USD）——NULL=无袋；
--   ② guard.bare_bag_min_usd=显形阈（残袋 USD 估值≥阈才落账——lotSz 碎尘免噪）；
--   ③ 已平仓仓腿账残袋回填：残袋=开腿实得 qty_base−平腿实卖 qty_base
--     （spot_sell cashflow source_ref 链回平腿行——两真源对拍非猜）；
--     settled_amount 按「现金净+挂账估值」重算=实现额非全损（幂等护栏：
--     bare_bag_qty 已立不重算；挂账估值=平腿卖价缺=开腿 px 兜底，皆缺=NULL
--     挂币量不挂虚值——消费面 COALESCE(0) 保守计）。
ALTER TABLE fin_position ADD COLUMN IF NOT EXISTS bare_bag_ccy text;
ALTER TABLE fin_position ADD COLUMN IF NOT EXISTS bare_bag_qty numeric;
ALTER TABLE fin_position ADD COLUMN IF NOT EXISTS bare_bag_usd numeric;
COMMENT ON COLUMN fin_position.bare_bag_ccy IS '平仓未回资产币种（T-0419 裸袋挂账——非 NULL=仓标 closed 但残袋在 venue）';
COMMENT ON COLUMN fin_position.bare_bag_qty IS '裸袋币量（原币）——腿账残袋=min(venue实余,开腿实得−平腿实卖)';
COMMENT ON COLUMN fin_position.bare_bag_usd IS '裸袋挂账估值 USD（检出时平腿卖价/开腿 px 口径——settled_amount=Σcashflow+本列=实现额；NULL=无价不挂虚值）';

INSERT INTO app_config(key,value) VALUES
  ('guard.bare_bag_min_usd','1')
ON CONFLICT (key) DO NOTHING;

WITH openq AS (
    SELECT p.id pid, l.inst_id, l.qty_base q, p.execution_id pexec
    FROM fin_position p
    JOIN fin_order_leg l ON l.execution_id=p.execution_id
    WHERE l.state='filled' AND l.side='buy' AND l.inst_id NOT LIKE '%-SWAP' AND l.qty_base IS NOT NULL
),
soldq AS (
    SELECT c.position_ref pid, l.inst_id, sum(l.qty_base) q,
           (array_agg(l.px ORDER BY l.ts DESC))[1] px
    FROM fin_cashflow c
    JOIN fin_order_leg l ON l.id=c.source_ref AND c.source_table='fin_order_leg'
    WHERE c.kind='spot_sell' AND l.qty_base IS NOT NULL
    GROUP BY 1,2
),
bag AS (
    SELECT o.pid, o.inst_id, o.q-COALESCE(s.q,0) qty,
           (o.q-COALESCE(s.q,0)) * COALESCE(s.px,
               (SELECT l3.px FROM fin_order_leg l3
                WHERE l3.execution_id=o.pexec AND l3.inst_id=o.inst_id AND l3.px IS NOT NULL
                ORDER BY l3.ts DESC LIMIT 1)) usd
    FROM openq o LEFT JOIN soldq s ON s.pid=o.pid AND s.inst_id=o.inst_id
    JOIN fin_position p2 ON p2.id=o.pid
    WHERE p2.closed_at IS NOT NULL AND o.q-COALESCE(s.q,0) > 1e-6
)
UPDATE fin_position p SET
    bare_bag_ccy=split_part(b.inst_id,'-',1),
    bare_bag_qty=b.qty,
    bare_bag_usd=b.usd,
    settled_amount=(SELECT COALESCE(sum(amount),0) FROM fin_cashflow
        WHERE position_ref=p.id AND kind NOT IN ('earn_in','earn_out')) + COALESCE(b.usd,0)
FROM bag b WHERE p.id=b.pid AND p.bare_bag_qty IS NULL AND p.settled_at IS NOT NULL;

-- 残袋已立但未结算仓只补挂账列（settle 岗结算时自加挂账额——同式不重算）。
WITH openq AS (
    SELECT p.id pid, l.inst_id, l.qty_base q, p.execution_id pexec
    FROM fin_position p
    JOIN fin_order_leg l ON l.execution_id=p.execution_id
    WHERE l.state='filled' AND l.side='buy' AND l.inst_id NOT LIKE '%-SWAP' AND l.qty_base IS NOT NULL
),
soldq AS (
    SELECT c.position_ref pid, l.inst_id, sum(l.qty_base) q,
           (array_agg(l.px ORDER BY l.ts DESC))[1] px
    FROM fin_cashflow c
    JOIN fin_order_leg l ON l.id=c.source_ref AND c.source_table='fin_order_leg'
    WHERE c.kind='spot_sell' AND l.qty_base IS NOT NULL
    GROUP BY 1,2
),
bag AS (
    SELECT o.pid, o.inst_id, o.q-COALESCE(s.q,0) qty,
           (o.q-COALESCE(s.q,0)) * COALESCE(s.px,
               (SELECT l3.px FROM fin_order_leg l3
                WHERE l3.execution_id=o.pexec AND l3.inst_id=o.inst_id AND l3.px IS NOT NULL
                ORDER BY l3.ts DESC LIMIT 1)) usd
    FROM openq o LEFT JOIN soldq s ON s.pid=o.pid AND s.inst_id=o.inst_id
    JOIN fin_position p2 ON p2.id=o.pid
    WHERE p2.closed_at IS NOT NULL AND o.q-COALESCE(s.q,0) > 1e-6
)
UPDATE fin_position p SET
    bare_bag_ccy=split_part(b.inst_id,'-',1),
    bare_bag_qty=b.qty,
    bare_bag_usd=b.usd
FROM bag b WHERE p.id=b.pid AND p.bare_bag_qty IS NULL AND p.settled_at IS NULL;

-- 事件簿显形（本迁落袋的仓每仓一行——幂等：同仓同票裸袋行不重复插）。
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '仓巡','裸袋','T-0419',
       format('仓#%s %s 残袋 %s≈$%s 挂账回填（平腿单位错配病灶——settled 实现额已按挂账口径重算）',
              p.id, p.bare_bag_ccy, p.bare_bag_qty,
              COALESCE(to_char(p.bare_bag_usd,'FM999990.00'),'无价'))
FROM fin_position p
WHERE p.bare_bag_qty IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM ops_event e WHERE e.event='裸袋' AND e.ticket='T-0419'
                  AND e.note LIKE '仓#'||p.id||' %');
