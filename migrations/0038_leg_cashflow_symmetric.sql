-- 0038 腿级现金流对称记账（T-0083——T-0082 错层矫正副作用根修）：
--   ① kind 闭集扩 spot_buy/spot_sell：fin_cashflow 保持现金流身份——现货腿
--      买=流出毛额、卖=流入毛额，两腿各记净额自然=真价差（trade_pnl 退役出
--      平仓径；损益=派生读面不改流水本体）。
--   ② 损益行出流水：source_table='fin_order_leg' 的 trade_pnl 行删除
--      （幻影盈/矫正产物——现金账只留现金流）。
--   ③ 历史回填：filled/rolled_back 现货腿（okx 非 SWAP）按 side 对称补行，
--      幂等锚 source_ref=腿 id（已记不重记）；毛额判式——sell=sz×px；
--      买：sz×px≤notional×0.75=base 委托毛额、sz×px 超阈=quote 委托（sz=USDT
--      额直取）、px 缺=名义半额兜底。
--   ④ fin_known_residual T-0083 行清场（残差归机械义零——无此行亦幂等）。
ALTER TABLE fin_cashflow DROP CONSTRAINT fin_cashflow_kind_check;
ALTER TABLE fin_cashflow ADD CONSTRAINT fin_cashflow_kind_check CHECK (kind IN
  ('fee','funding_settle','trade_pnl','rollback_cost','deposit','withdraw',
   'earn_in','earn_out','interest','recon_adjust','spot_buy','spot_sell'));

DELETE FROM fin_cashflow WHERE kind='trade_pnl' AND source_table='fin_order_leg';

INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
SELECT CASE WHEN l.side='sell' THEN 'spot_sell' ELSE 'spot_buy' END,
       l.venue, 'USDT',
       CASE WHEN l.side='sell' THEN
              CASE WHEN l.px>0 THEN l.sz*l.px ELSE p.notional_usd/2 END
            ELSE -(CASE
              WHEN l.px>0 AND l.sz*l.px <= p.notional_usd*0.75 THEN l.sz*l.px
              WHEN l.px>0 THEN l.sz
              ELSE p.notional_usd/2 END) END,
       p.id, l.id, 'fin_order_leg', l.ts
FROM fin_order_leg l JOIN fin_position p ON p.execution_id=l.execution_id
WHERE l.state IN ('filled','rolled_back') AND l.venue='okx' AND l.inst_id NOT LIKE '%-SWAP'
  AND NOT EXISTS(SELECT 1 FROM fin_cashflow c
    WHERE c.source_table='fin_order_leg' AND c.source_ref=l.id);

DO $$ BEGIN
  IF to_regclass('fin_known_residual') IS NOT NULL
  THEN DELETE FROM fin_known_residual WHERE ticket_ref='T-0083'; END IF;
END $$;
