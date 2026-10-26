-- 0040 仓行归属修复（T-0083 返修③④）：
--   病灶：close exec 曾经 ⑥ 无条件建 fin_position 空壳行（execution_id=close
--   exec），0038 回填 JOIN p.execution_id=l.execution_id 把 close 卖腿现金行
--   绑到壳行而非被平的原开仓——position_ref 归属错。
--   ③④ 修法：卖腿现金行 position_ref 改挂「该 inst 最近已开仓」（非壳行，
--   opened_at<=腿时点+1h 防荒配）；壳行在零 cashflow 引用后删除。
--   壳行判式：execution 名下存在 okx 现货 sell filled 腿（open 径现货腿只有
--   buy；LEGGED 回滚卖腿=rolled_back 态不命中）。幂等：重绑后壳引用清零，
--   重跑零行；新径（close 不建壳）产出零壳行同样零命中。
--   定位口径：sell 腿 source_ref 现金行 position_ref∈壳集才重绑（NULL 行不猜）。

UPDATE fin_cashflow c
SET position_ref = (
  SELECT o.id FROM fin_position o
  WHERE o.inst_id = (SELECT sh.inst_id FROM fin_position sh WHERE sh.id = c.position_ref)
    AND NOT EXISTS (
      SELECT 1 FROM fin_order_leg sl
      WHERE sl.execution_id = o.execution_id AND sl.side = 'sell'
        AND sl.state = 'filled' AND sl.venue = 'okx' AND sl.inst_id NOT LIKE '%-SWAP')
    AND o.opened_at <= c.occurred_at + interval '1 hour'
  ORDER BY o.id DESC LIMIT 1)
WHERE c.source_table = 'fin_order_leg'
  AND c.position_ref IN (
    SELECT p.id FROM fin_position p
    WHERE EXISTS (
      SELECT 1 FROM fin_order_leg l
      WHERE l.execution_id = p.execution_id AND l.side = 'sell'
        AND l.state = 'filled' AND l.venue = 'okx' AND l.inst_id NOT LIKE '%-SWAP'));

DELETE FROM fin_position p
WHERE EXISTS (
  SELECT 1 FROM fin_order_leg l
  WHERE l.execution_id = p.execution_id AND l.side = 'sell'
    AND l.state = 'filled' AND l.venue = 'okx' AND l.inst_id NOT LIKE '%-SWAP')
  AND NOT EXISTS (SELECT 1 FROM fin_cashflow c WHERE c.position_ref = p.id);
