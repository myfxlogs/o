-- 0124 T-0405 面清单快照窄表——饥饿拍每拍机器可列面（服务/表/产消对/
--   模板/闸键域）落批行，与上拍 diff 新面=升件（「该建未建」类残余：
--   探针查产消对不查「系统该有什么」，对面清单快照+diff 退化为可见差集行）。
--   窄表不建重型件——批轴=batch（ops_face_batch_seq 单调派号），仅留最近
--   两拍批（diff 毕旧批即清，探针自清非保留期件）。写方=lifecycle hunger
--   拍（facesnap.go）；读方=审计/COS 收件箱面。
CREATE SEQUENCE IF NOT EXISTS ops_face_batch_seq;

CREATE TABLE IF NOT EXISTS ops_face_snapshot (
    id    bigserial PRIMARY KEY,
    batch bigint NOT NULL,               -- 拍批号（ops_face_batch_seq——同批一面一行）
    ts    timestamptz NOT NULL DEFAULT now(),
    kind  text NOT NULL,                 -- 面类：service|table|dep_pair|template|gate_domain
    face  text NOT NULL                  -- 面键（seat/表名/对签名/template/键域）
);
COMMENT ON TABLE ops_face_snapshot IS
  '面清单快照窄表（T-0405）——机器可列面每拍落批+上拍 diff 新面升件；仅留最近两拍批';
CREATE INDEX IF NOT EXISTS ops_face_snapshot_batch ON ops_face_snapshot(batch,kind);

INSERT INTO fin_data_retention(table_name,class,note) VALUES
  ('ops_face_snapshot','state','面清单快照仅留最近两拍批（T-0405——探针自清，非保留期件）')
ON CONFLICT (table_name) DO NOTHING;

-- 消费面授权（条件式——角色缺席不炸；fin_0123 同例）：
--   roster_rw=dev 审计/对账窄面；o_fin_ro=COS 收件箱新面可视。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT SELECT,INSERT,DELETE ON ops_face_snapshot TO roster_rw';
		EXECUTE 'GRANT USAGE ON SEQUENCE ops_face_snapshot_id_seq,ops_face_batch_seq TO roster_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
		EXECUTE 'GRANT SELECT ON ops_face_snapshot TO o_fin_ro';
	END IF;
END $$;
