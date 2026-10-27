-- 0066 md_funding 投影幂等键（T-0167/D-0032 §三-2 裁①：UNIQUE(venue,inst_id,ts)
-- 双链同加——超表唯一索引含分区列 ts 可建；保历史/保回滚/幂等三收）。
-- 先去重再落键：现库共写双源（dev collector+SG 本地）同拍重复行实证在案，
-- 保留 recv_ts 最新行（D-0018 同键最新赢语义）；键后 projector/共写方
-- ON CONFLICT 幂等可写（同 ts 同拍=同值 DO NOTHING），回滚径「DDL 相同」
-- 前提由 check_migrations 桥表归一化断言（含索引集）机检钉死。幂等可重放。
-- 迁移原子性（T-0169 轮换实证）：结算拍写流活跃期 DELETE 与 CREATE INDEX 两
-- 语句间 READ COMMITTED 新快照可见并发重复行=建索引 23505（首转实挫回滚）。
-- SHARE ROW EXCLUSIVE 阻并发写（读不阻），去重+建键同锁内原子完成；写方批次
-- 阻塞秒级让位（新码 ON CONFLICT 后键在无感）。
LOCK TABLE md_funding IN SHARE ROW EXCLUSIVE MODE;
DELETE FROM md_funding a
  USING md_funding b
  WHERE a.venue=b.venue AND a.inst_id=b.inst_id AND a.ts=b.ts
    AND (a.recv_ts, a.ctid) < (b.recv_ts, b.ctid);
CREATE UNIQUE INDEX IF NOT EXISTS md_funding_venue_inst_ts_uq
  ON md_funding (venue, inst_id, ts);
