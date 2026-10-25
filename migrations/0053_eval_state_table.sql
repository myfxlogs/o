-- 0051 T-0126 fin_opp_eval 状态表化（紧急——25k 行/分≈9GB/日烧穿磁盘预警）：
--   评估=状态非流水——同标的同模板最新判定原地覆盖（D-0018 同型）。
--   ① 提案 eval_id 重指键域最新行（提案行自带 inst/direction/net_apr——
--     重指不丢业务语义）→ 非最新行删 → 唯一键落；
--   ② ev_eval 触发器扩 UPDATE——upsert 覆盖仍唤醒 lifecycle（不然
--     rejected→proposed 翻牌无事件=晋升链断流）。
WITH keep AS (
    SELECT inst_id, template, direction, max(id) AS keep_id
    FROM fin_opp_eval GROUP BY 1, 2, 3
)
UPDATE fin_proposal p SET eval_id = k.keep_id
FROM fin_opp_eval e
JOIN keep k ON k.inst_id=e.inst_id AND k.template=e.template AND k.direction=e.direction
WHERE p.eval_id = e.id AND e.id <> k.keep_id;

DELETE FROM fin_opp_eval e
WHERE EXISTS (
    SELECT 1 FROM fin_opp_eval k
    WHERE k.inst_id=e.inst_id AND k.template=e.template AND k.direction=e.direction
      AND k.id > e.id)
  AND NOT EXISTS (SELECT 1 FROM fin_proposal p WHERE p.eval_id = e.id);

ALTER TABLE fin_opp_eval
    ADD CONSTRAINT fin_opp_eval_key UNIQUE (inst_id, template, direction);

DROP TRIGGER IF EXISTS ev_eval_trg ON fin_opp_eval;
CREATE TRIGGER ev_eval_trg AFTER INSERT OR UPDATE ON fin_opp_eval
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_eval');
