-- 0128 T-0412 意图晋级径（提醒/自选建单即订阅晋升）fin 链：
--   ① ev_subreq 触发器——requester='user_intent' 行变即 NOTIFY（projector
--     反向桥即发径；ROW 级 WHEN 滤域——正向桥写非 user_intent 行不扰，
--     防每帧空醒；ev_notify 沿用 0028 事件脊函数不新造范式）；
--   ② 存量僵尸修复：fin_price_alert 未终态行符号在目录（md_tick_scan 活
--     符）而无活 tick→user_intent open 行一次性复活。票面钦「迁移或建单
--     重放择其一」——判词=迁移径：数据驱动逐行覆盖全部同型僵尸非只 #1
--     TRX（同事物同判不留个案缝）。
CREATE OR REPLACE TRIGGER ev_subreq_trg AFTER INSERT OR UPDATE ON fin_sub_request
  FOR EACH ROW WHEN (NEW.requester='user_intent')
  EXECUTE FUNCTION ev_notify('ev_subreq');

INSERT INTO fin_sub_request(venue,inst_id,requester,state,reason)
SELECT 'okx', a.symbol, 'user_intent', 'open',
       'T-0412 存量僵尸复活：fin_price_alert #'||a.id
FROM fin_price_alert a
WHERE a.state IN ('active','disabled')
  AND NOT EXISTS (SELECT 1 FROM md_ticks t WHERE t.inst_id=a.symbol
    AND t.recv_ts > now()-CASE t.venue
      WHEN 'mt5' THEN interval '72 hours'
      ELSE interval '15 minutes' END)
  AND EXISTS (SELECT 1 FROM md_tick_scan s WHERE s.inst_id=a.symbol
    AND s.recv_ts > now()-interval '1 hour')
ON CONFLICT (venue,inst_id,requester) DO UPDATE SET state='open',
    reason=EXCLUDED.reason,ts=now();
