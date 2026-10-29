-- 1075 arch_eval 评估历史载（T-0230 Phase-0/D-0036 §12.5 序3——fin_opp_eval
-- 现态覆盖写不可逆损根治）：append-only+change-only（同键值重复跳）。
-- 选型=触发器：写径 6 点跨 3 包（dcd/grammar/funding/gapscan/hypothesis/
-- projector mirror）——逐点同事务改判=漏点险+双写放大险；触发器内 change
-- 门天然幂等且未来写径零漏接（D-0036 §6「无写径点才落触发器」——本面=
-- 写径弥散型，单点闸优于多点改）。AFTER 触发器失败=主事务回滚——arch 写
-- 用单句 INSERT 且仅读自身表，失败域=DDL 级，旁路告警由哨兵/水印闸兜。
CREATE TABLE IF NOT EXISTS arch_eval (
	arch_id        bigserial PRIMARY KEY,
	inst_id        text NOT NULL,
	template       text NOT NULL,
	direction      text NOT NULL,
	verdict        text NOT NULL,
	gross_apr      numeric,
	net_apr        numeric,
	legs           jsonb NOT NULL,
	evidence       jsonb NOT NULL,
	src_ts         timestamptz NOT NULL,
	arch_ts        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS arch_eval_key ON arch_eval(inst_id,template,direction,arch_id);
COMMENT ON TABLE arch_eval IS 'fin_opp_eval 变化序列历史载（change-only append——D-0018 修订：arch_* 为唯一合法历史载体）';

CREATE OR REPLACE FUNCTION arch_eval_capture() RETURNS trigger AS $$
DECLARE last_r record;
BEGIN
	SELECT verdict,gross_apr,net_apr,legs::text l,evidence::text e INTO last_r
		FROM arch_eval WHERE inst_id=NEW.inst_id AND template=NEW.template
		AND direction=NEW.direction ORDER BY arch_id DESC LIMIT 1;
	IF FOUND
		AND last_r.verdict IS NOT DISTINCT FROM NEW.verdict
		AND last_r.gross_apr IS NOT DISTINCT FROM NEW.gross_apr
		AND last_r.net_apr IS NOT DISTINCT FROM NEW.net_apr
		AND last_r.l IS NOT DISTINCT FROM NEW.legs::text
		AND last_r.e IS NOT DISTINCT FROM NEW.evidence::text THEN
		RETURN NULL; -- 同键同值=跳（change-only 门）
	END IF;
	INSERT INTO arch_eval(inst_id,template,direction,verdict,gross_apr,net_apr,legs,evidence,src_ts)
		VALUES(NEW.inst_id,NEW.template,NEW.direction,NEW.verdict,NEW.gross_apr,
			NEW.net_apr,NEW.legs,NEW.evidence,NEW.ts);
	RETURN NULL;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS arch_eval_trg ON fin_opp_eval;
CREATE TRIGGER arch_eval_trg AFTER INSERT OR UPDATE ON fin_opp_eval
	FOR EACH ROW EXECUTE FUNCTION arch_eval_capture();
