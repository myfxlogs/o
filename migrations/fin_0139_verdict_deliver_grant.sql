-- 0139 T-0438 判词送达闸箱写授权（fin_0113 research_rw 同形——roster_rw
-- 补授）：判词送达面（scripts/meta/verdict_deliver.sh，dm_tick 机械轨尾窗
-- 载位）需业主动作判词→fin_condition_gap 请求箱转写。FIN_PG_DSN=roster_rw
-- 对箱零授权实勘（SG 正本）——按 fin_0113 既有柱级口径补授：INSERT 全表+
-- SELECT 限列（id/kind/venue/surface/payload/state/requested_at——RETURNING
-- id 幂等留痕+箱读面）+UPDATE 限列（requested_at,payload——condreq 刷新
-- 语义）；**不授 state/resolved_at/resolve_note 更新权**——业主处置仍走
-- 既有窄写面（fin_0113 钦定不变）。送达标记行写 ops_event（roster_rw
-- 既有 INSERT 权，fin_0086）零新授权。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'roster_rw') THEN
    GRANT INSERT ON fin_condition_gap TO roster_rw;
    GRANT SELECT (id,kind,venue,surface,payload,state,requested_at) ON fin_condition_gap TO roster_rw;
    GRANT UPDATE (requested_at,payload) ON fin_condition_gap TO roster_rw;
    GRANT USAGE ON SEQUENCE fin_condition_gap_id_seq TO roster_rw;
  END IF;
END $$;
