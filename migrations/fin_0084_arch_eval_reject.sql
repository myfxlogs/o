-- 0084 arch_eval_reject 拒件历史载（T-0230——fin_eval_reject 现态表变化序列
-- 不可重建根治；触发器选型同 arch_eval——写径虽单点（lifecycle rejectEvalReason），
-- 触发器同构免新写径型，change 门统一语义）。
CREATE TABLE IF NOT EXISTS arch_eval_reject (
	arch_id        bigserial PRIMARY KEY,
	inst_id        text NOT NULL,
	template       text NOT NULL,
	direction      text NOT NULL,
	reason         text NOT NULL,
	evidence       jsonb NOT NULL,
	src_ts         timestamptz NOT NULL,
	arch_ts        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS arch_eval_reject_key ON arch_eval_reject(inst_id,template,direction,reason,arch_id);
COMMENT ON TABLE arch_eval_reject IS 'fin_eval_reject 变化序列历史载（change-only append——T-0230）';

CREATE OR REPLACE FUNCTION arch_eval_reject_capture() RETURNS trigger AS $$
DECLARE last_r record;
BEGIN
	SELECT reason,evidence::text e INTO last_r FROM arch_eval_reject
		WHERE inst_id=NEW.inst_id AND template=NEW.template
		AND direction=NEW.direction AND reason=NEW.reason
		ORDER BY arch_id DESC LIMIT 1;
	IF FOUND AND last_r.e IS NOT DISTINCT FROM NEW.evidence::text THEN
		RETURN NULL;
	END IF;
	INSERT INTO arch_eval_reject(inst_id,template,direction,reason,evidence,src_ts)
		VALUES(NEW.inst_id,NEW.template,NEW.direction,NEW.reason,NEW.evidence,NEW.ts);
	RETURN NULL;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS arch_eval_reject_trg ON fin_eval_reject;
CREATE TRIGGER arch_eval_reject_trg AFTER INSERT OR UPDATE ON fin_eval_reject
	FOR EACH ROW EXECUTE FUNCTION arch_eval_reject_capture();
