-- 0066 md_funding 投影幂等键（T-0167/D-0032 §三-2 裁①：UNIQUE(venue,inst_id,ts)
-- 双链同加——超表唯一索引含分区列 ts 可建；保历史/保回滚/幂等三收）。
-- 先去重再落键：现库共写双源（dev collector+SG 本地）同拍重复行实证在案，
-- 保留 recv_ts 最新行（D-0018 同键最新赢语义）；键后 projector/共写方
-- ON CONFLICT 幂等可写（同 ts 同拍=同值 DO NOTHING），回滚径「DDL 相同」
-- 前提由 check_migrations 桥表归一化断言（含索引集）机检钉死。幂等可重放。
DELETE FROM md_funding a
  USING md_funding b
  WHERE a.venue=b.venue AND a.inst_id=b.inst_id AND a.ts=b.ts
    AND (a.recv_ts, a.ctid) < (b.recv_ts, b.ctid);
CREATE UNIQUE INDEX IF NOT EXISTS md_funding_venue_inst_ts_uq
  ON md_funding (venue, inst_id, ts);
