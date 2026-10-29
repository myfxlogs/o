-- 1076 arch_signal 信号历史载+md_daily_snapshot 活跃集日快照表（T-0230——
-- md_signal md_1057 禁时序不可逆损根治同型；快照表体在册 cron 填）。
CREATE TABLE IF NOT EXISTS arch_signal (
	arch_id        bigserial PRIMARY KEY,
	kind           text NOT NULL,
	key            text NOT NULL,
	val            numeric,
	meta           jsonb NOT NULL DEFAULT '{}',
	src_ts         timestamptz NOT NULL,
	arch_ts        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS arch_signal_key ON arch_signal(kind,key,arch_id);
COMMENT ON TABLE arch_signal IS 'md_signal 变化序列历史载（change-only append——T-0230）';

CREATE TABLE IF NOT EXISTS md_daily_snapshot (
	id             bigserial PRIMARY KEY,
	snap_date      date NOT NULL,
	inst_id        text NOT NULL,
	last           numeric,
	src            text NOT NULL, -- tick|mark_index
	captured_at    timestamptz NOT NULL DEFAULT now(),
	UNIQUE (snap_date, inst_id, src)
);
COMMENT ON TABLE md_daily_snapshot IS '活跃集行情日快照（T-0230 archive-only 显式标注——非业务刷新面；日拍 cron 填，arch_keep_days 保留期闸同族）';

CREATE OR REPLACE FUNCTION arch_signal_capture() RETURNS trigger AS $$
DECLARE last_r record;
BEGIN
	SELECT val,meta::text m INTO last_r FROM arch_signal
		WHERE kind=NEW.kind AND key=NEW.key ORDER BY arch_id DESC LIMIT 1;
	IF FOUND
		AND last_r.val IS NOT DISTINCT FROM NEW.val
		AND last_r.m IS NOT DISTINCT FROM NEW.meta::text THEN
		RETURN NULL;
	END IF;
	INSERT INTO arch_signal(kind,key,val,meta,src_ts)
		VALUES(NEW.kind,NEW.key,NEW.val,NEW.meta,NEW.ts);
	RETURN NULL;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS arch_signal_trg ON md_signal;
CREATE TRIGGER arch_signal_trg AFTER INSERT OR UPDATE ON md_signal
	FOR EACH ROW EXECUTE FUNCTION arch_signal_capture();
