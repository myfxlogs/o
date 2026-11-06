-- 0123 T-0393 探针升件→自动起草修复票管——候选票草案锚表：
-- 起草拍（internal/portfolio/hunger drafter.go，hungerTick 同槽）扫
-- ops_event 探针升件行→病灶签名（kind=esc_class 探针类+key=esc_scope
-- 病灶键）聚类→签名级去重（同键 24h 防灌/同键在飞·已立票不重复起草）
-- →pending 草案行落本表；物料化（tools/ticketdraft，dm_tick 零 LLM 尾段）
-- 取号（next_ticket.sh 唯一径）→docs/tasks 候选态票件→status=materialized
-- +ticket 落号。写径只落草案态零自派——发起权=DM/业主（T-0376 权序红线）。
CREATE TABLE IF NOT EXISTS fin_ticket_draft (
    id         bigserial PRIMARY KEY,
    ts         timestamptz NOT NULL DEFAULT now(),
    signature  text NOT NULL,               -- 病灶签名（kind:key 等值键）
    kind       text NOT NULL,               -- 探针类（升件 esc_class）
    lesion_key text NOT NULL,               -- 病灶键（升件 esc_scope）
    status     text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','materialized','dismissed')),
    payload    jsonb NOT NULL DEFAULT '{}',  -- 物料化件（slug/title/evidence_ids/
                                           --  evidence_note/evidence_query/domain_hint）
    ticket     text NOT NULL DEFAULT '',     -- 物料化落号（pending 期空）
    materialized_at timestamptz
);
COMMENT ON TABLE fin_ticket_draft IS
  '候选票草案锚表（T-0393）——探针升件签名聚类 pending→物料化 docs/tasks 候选票（草案态零自派）';
CREATE INDEX IF NOT EXISTS fin_ticket_draft_sig_ts ON fin_ticket_draft(signature,ts DESC);
CREATE INDEX IF NOT EXISTS fin_ticket_draft_pending ON fin_ticket_draft(id) WHERE status='pending';
INSERT INTO fin_data_retention(table_name,class,note) VALUES
  ('fin_ticket_draft','state','候选票草案锚表不删行（T-0393）')
ON CONFLICT (table_name) DO NOTHING;
-- 消费面授权（条件式——角色缺席不炸；fin_0120 同例）：
--   roster_rw=dev 物料化/裁写窄面（pending 读+status/ticket 落写）；
--   o_fin_ro 窄读面（COS 收件箱草案可视）。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT SELECT,INSERT,UPDATE ON fin_ticket_draft TO roster_rw';
		EXECUTE 'GRANT USAGE ON SEQUENCE fin_ticket_draft_id_seq TO roster_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
		EXECUTE 'GRANT SELECT ON fin_ticket_draft TO o_fin_ro';
	END IF;
END $$;
