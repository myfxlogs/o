-- fin_0088 无头单状态面（T-0263 业主令——「不看内容只要知道活着」推送制信号径）。
-- ops_headless=每票一行有界态（D-0018 语义同 fin_host_metrics：PK(ticket) upsert
-- 覆写零增长；非 venue 快照——本系统席位活性自报遥测）。写方=dev 泵
-- cmd/headlesspump（inotify 日志写/.done/.hb 心跳+泵侧静默超时判死挂——
-- ROSTER_PG_DSN 隧道 research_rw 径）；相位源=ops_event 席自报（轻碰触发器
-- 并入同一事件轴）。seq=单调行版（BEFORE 触发器派 nextval——前端 unary 快照
-- +流帧 seq 去重补洞的水印轴）。
CREATE SEQUENCE IF NOT EXISTS ops_headless_seq;

CREATE TABLE IF NOT EXISTS ops_headless (
    ticket        text PRIMARY KEY,          -- T-NNNN 票号
    seq           bigint NOT NULL DEFAULT 0, -- 行版（触发器派 nextval——客户端去重水印轴）
    seat          text NOT NULL DEFAULT '',  -- 席位（Dev=施工 run / DM=审计 audit——日志前缀派生）
    model         text NOT NULL DEFAULT '',  -- 底模（/proc environ DEVIN_MODEL 实读——读件缺=空）
    state         text NOT NULL DEFAULT '',  -- 五态机键（见列注）
    last_activity timestamptz,               -- 末次活动=日志 mtime（.hb 触碰同计座席活证）
    updated_at    timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE ops_headless IS
  '无头单状态面有界态（T-0263——每票一行 upsert；写方=dev 泵 cmd/headlesspump；读方=o-core GetHeadless+Watch ev_headless）';
COMMENT ON COLUMN ops_headless.state IS
  '五态机键：alive在飞|stalled停滞(日志静默>30min进程在)|dead死(进程殁未完工)|overdue逾期(在飞>8h)|done完工——服务端判态前端零判据';
COMMENT ON COLUMN ops_headless.seq IS '行版单调序号（ops_headless_seq——流帧/快照去重补洞水印轴）';

-- 行版派号（每次写必升——相位轻碰同样升版进轴）。
CREATE OR REPLACE FUNCTION ops_headless_stamp() RETURNS trigger AS $$
BEGIN NEW.seq := nextval('ops_headless_seq'); RETURN NEW; END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE TRIGGER ops_headless_seq_trg
BEFORE INSERT OR UPDATE ON ops_headless
FOR EACH ROW EXECUTE FUNCTION ops_headless_stamp();

-- 状态写→事件帧（语句级 ev_notify 复用——纯唤醒语义不携值，读面现查防陈拍）。
CREATE OR REPLACE TRIGGER ev_headless_trg
AFTER INSERT OR UPDATE ON ops_headless
FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_headless');

-- 相位变更（席自报 ops_event）并入同一事件轴：轻碰在册行——ops_headless 自身
-- BEFORE UPDATE 派 seq + STATEMENT 触发器发 ev_headless，单 seq 轴单 channel
-- （未在册票=0 行轻碰=无事件——泵建档后相位 join 现值自愈）。
CREATE OR REPLACE FUNCTION ops_event_touch_headless() RETURNS trigger AS $$
BEGIN
  UPDATE ops_headless SET updated_at = now() WHERE ticket = NEW.ticket;
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE TRIGGER ops_event_touch_trg
AFTER INSERT ON ops_event
FOR EACH ROW EXECUTE FUNCTION ops_event_touch_headless();

-- 写方授权（research_rw=ROSTER_PG_DSN 隧道径——与 report.sh ops_event 写径同源；
-- 泵 upsert 用 INSERT+UPDATE，seq 消耗须 USAGE；角色缺席环境=零失败同族规）。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'research_rw') THEN
    GRANT SELECT, INSERT, UPDATE ON ops_headless TO research_rw;
    GRANT USAGE ON SEQUENCE ops_headless_seq TO research_rw;
  END IF;
END $$;
