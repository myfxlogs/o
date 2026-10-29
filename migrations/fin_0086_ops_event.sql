-- 0086 席自报落表（T-0239 业主驾驶舱数据源——append-only 事件簿）：
-- report.sh 双写径（飞书推既有不动+本表插行）；视图=每席/每票最新派生。
CREATE TABLE IF NOT EXISTS ops_event (
	id       bigserial PRIMARY KEY,
	ts       timestamptz NOT NULL DEFAULT now(),
	seat     text NOT NULL,
	event    text NOT NULL,
	ticket   text NOT NULL DEFAULT '',
	note     text NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS ops_event_seat_ts ON ops_event(seat,ts DESC);
COMMENT ON TABLE ops_event IS '席自报事件簿（T-0239 驾驶舱存活面/工单板/阶段面数据源——append-only）';

CREATE OR REPLACE VIEW ops_seat_latest AS
	SELECT DISTINCT ON (seat) seat,ts,event,ticket,note
	FROM ops_event ORDER BY seat,ts DESC;
CREATE OR REPLACE VIEW ops_ticket_state AS
	SELECT DISTINCT ON (ticket) ticket,event,ts,seat
	FROM ops_event WHERE ticket<>'' ORDER BY ticket,ts DESC;

-- research_rw 自报写径（report.sh 双写=research_rw DSN——INSERT 表权+序权，
-- 读面 SELECT 三视图——表随册可复现非仅产侧手工授）。
GRANT INSERT ON ops_event TO research_rw;
GRANT USAGE ON SEQUENCE ops_event_id_seq TO research_rw;
GRANT SELECT ON ops_event,ops_seat_latest,ops_ticket_state TO research_rw;
