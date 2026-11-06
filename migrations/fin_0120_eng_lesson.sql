-- 0120 T-0385 教训矿工拍——工程案锚表（事件账自动挖工程案 pending）：
-- 经济案 fin_econ_lesson 机件同构复用，工程案独立成表不挤 kind CHECK 集
-- （票面「扩 kind='eng' 或新 fin_eng_lesson——工人按面谱择」择新表：
--   econ 载荷=econv1 protojson 契约面，eng 载荷=签名/证据锚/草案/条目名/
--   验证态机读五件——异谱共表=两案互染，专用表列级面自明）。
-- signature=矿工聚类签名（升件=esc_class+esc_scope／驳回·异常·返修=票号
-- 维度键）；verify_state 四档钦定（auto_verified/ai_evaluated/human_
-- confirmed/unverified——「没验」与「验过」显态分位，矿工起草落
-- ai_evaluated：机器聚类稿带实证锚非未验，亦未过人工非 confirmed）。
-- 矿工只挖不裁——status pending→verdicted 归裁定席既有无头裁流（verdict
-- 行照旧三态 canon/dismiss/merge，KB 条目落 docs/工程经验库）。
CREATE TABLE IF NOT EXISTS fin_eng_lesson (
    id           bigserial PRIMARY KEY,
    ts           timestamptz NOT NULL DEFAULT now(),
    signature    text NOT NULL,              -- 聚类签名（矿工签名级去重键）
    cluster_n    int  NOT NULL DEFAULT 1,    -- 窗内同签名证据行数
    verify_state text NOT NULL DEFAULT 'ai_evaluated'
        CHECK (verify_state IN
            ('auto_verified','ai_evaluated','human_confirmed','unverified')),
    numbers      jsonb NOT NULL DEFAULT '{}', -- 机读五件（evidence_ids/draft/
                                             --  suggested_entry/tickets/event）
    status       text NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending','verdicted','canon')),
    verdict_ref  text NOT NULL DEFAULT ''
);
COMMENT ON TABLE fin_eng_lesson IS
  '工程案锚表（T-0385）——ops_event 窗聚类 pending→裁定席 verdict→工程经验库条目';
COMMENT ON COLUMN fin_eng_lesson.verify_state IS
  '验证态四档（博主指南——auto_verified/ai_evaluated/human_confirmed/unverified；矿工起草带实证锚落 ai_evaluated）';
CREATE INDEX IF NOT EXISTS fin_eng_lesson_sig_ts ON fin_eng_lesson(signature,ts DESC);
CREATE INDEX IF NOT EXISTS fin_eng_lesson_pending ON fin_eng_lesson(id) WHERE status='pending';
INSERT INTO fin_data_retention(table_name,class,note) VALUES
  ('fin_eng_lesson','state','工程案锚表不删行（T-0385）')
ON CONFLICT (table_name) DO NOTHING;
-- 消费面授权（条件式——角色缺席不炸，全新部署哑死防；fin_0103 同例）：
--   roster_rw=dev 裁定席隧道凭据（pending 读+verdict_ref/status 裁写窄面）；
--   o_fin_ro 窄读面（COS 收件箱工程案可视——fin_0111 同例）。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT SELECT,UPDATE ON fin_eng_lesson TO roster_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
		EXECUTE 'GRANT SELECT ON fin_eng_lesson TO o_fin_ro';
	END IF;
END $$;
