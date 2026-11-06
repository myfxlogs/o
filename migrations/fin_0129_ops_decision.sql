-- 0129 T-0415 决策记忆表——钱径判据与席位裁词同构落账。
-- 病灶（票面实锤）：fin_strategy 有 retire_reason 列而仓位「为什么平」零凭据
--   （ops_event 零平仓事件）、DM 裁决落票「通过 merge sha」零判词——拒的有案、
--   活的没案、死的没墓志铭。记忆=结构账非相似检索（pgvector 预留位不立）。
-- 四件：
--   ① ops_decision 表——钦定语义 (domain,actor,ref_id,action,reason,payload,ts)；
--     词集闭集表内 CHECK：fin=open|close|adjust|reject、seat=裁决|驳回|参数调整|
--     升件裁量；reason 禁空（裸行结构性不可落）。
--   ② ops_decision_write() 库内写口单源——psql/壳/触发器/Go 同径必经
--     （写面异构无单一 app 层收口点=fin_0121 同判据）；seat 裁词行裸词
--     （通过/驳回/升级/裁决 孤词）=RAISE——红桩②「裸行绝迹」机械钉死：
--     裸词裁词在任何写径下都不可落库。
--   ③ 镜像触发器两件：
--     a) ops_event AFTER INSERT——seat 裁词三词（裁决/驳回/参数调整）全径镜像
--        （report.sh psql/DMSet 同事务行/手工 psql 异构写面必经——dmwrite
--        同径钦定即此义：事件行落账同拍 decision 行落账）；COS 升件打包位
--        留写口=COS上行行+seat='COS' 升件行→升件裁量。
--     b) fin_econ_lesson/fin_eng_lesson AFTER UPDATE status→verdicted——
--        裁定席 verdict 同写（reason=verdict_ref 判词行；裸空=RAISE 令
--        verdict UPDATE 失败=裁席见错重写非无声漏账）。
--   ④ 授权同 ops_event 口径（research_rw/report.sh、review_rw/OCI 臂、
--     roster_rw/服务面 写+读+序权；o_fin_ro 读——镜像触发器 INVOKER 语义下
--     事件行写者须持 ops_decision 写权，幂等重授保全部署窗）。

CREATE TABLE IF NOT EXISTS ops_decision (
	id       bigserial PRIMARY KEY,
	ts       timestamptz NOT NULL DEFAULT now(),
	domain   text NOT NULL CHECK (domain IN ('fin','seat')),
	actor    text NOT NULL,
	ref_id   text NOT NULL DEFAULT '',
	action   text NOT NULL,
	reason   text NOT NULL,
	payload  jsonb,
	CHECK (btrim(reason) <> ''),
	CHECK ((domain='fin'  AND action IN ('open','close','adjust','reject'))
	    OR (domain='seat' AND action IN ('裁决','驳回','参数调整','升件裁量')))
);
CREATE INDEX IF NOT EXISTS ops_decision_domain_ts ON ops_decision(domain,ts DESC);
CREATE INDEX IF NOT EXISTS ops_decision_ref ON ops_decision(ref_id,ts DESC);
COMMENT ON TABLE ops_decision IS '决策记忆账（T-0415——钱径判据+席位裁词同构；fin=机器域件开/平/调/拒、seat=席位裁词；append-only 问责底座，消费面 strategy-review 预留）';
COMMENT ON COLUMN ops_decision.ref_id IS 'fin=strategy/proposal id；seat=票号/案号（lesson 镜像=表名#id）';
COMMENT ON COLUMN ops_decision.reason IS '触发判据+实况（flip_bps=xx/超龄72h/期望carry）|判词摘要（凭什么）——非空+seat 裁词禁裸词（写口函数守）';
COMMENT ON COLUMN ops_decision.payload IS '机读件——平仓行必携 expected_net_apr/realized_usd（实现的vs期望的）';

-- ② 写口函数（词集/裸词校验库内单源；ref/reason 规范化同口）。
CREATE OR REPLACE FUNCTION ops_decision_write(
	p_domain text, p_actor text, p_ref text, p_action text,
	p_reason text, p_payload jsonb DEFAULT NULL)
RETURNS bigint AS $$
DECLARE
	r   text := btrim(COALESCE(p_reason,''));
	nid bigint;
BEGIN
	IF p_domain='seat' AND p_action IN ('裁决','驳回')
	   AND r IN ('通过','驳回','升级','裁决') THEN
		RAISE EXCEPTION 'ops_decision 裁词行禁裸词——reason 必携判词摘要（T-0415 红桩②）';
	END IF;
	INSERT INTO ops_decision(domain,actor,ref_id,action,reason,payload)
		VALUES(p_domain,p_actor,COALESCE(p_ref,''),p_action,r,p_payload)
		RETURNING id INTO nid;
	RETURN nid;
END;
$$ LANGUAGE plpgsql;
COMMENT ON FUNCTION ops_decision_write(text,text,text,text,text,jsonb) IS
	'ops_decision 唯一写口（T-0415——seat 裁词裸词 RAISE；CHECK 词集兜底；席位/脚本/镜像触发器同径）';

-- ③a ops_event 镜像（seat 裁词全径镜像+COS 升件打包写口）。
-- 部署窗回落：表/写口未落产窗期 undefined_*=跳过镜像事件照落（fin_0111
-- 「列未随迁移落产窗期回落」同款——记忆缺行>吞事件；裸词 RAISE 非本类不受吞）。
CREATE OR REPLACE FUNCTION ops_decision_mirror() RETURNS trigger AS $$
BEGIN
	BEGIN
		IF NEW.event IN ('裁决','驳回','参数调整') THEN
			PERFORM ops_decision_write('seat',NEW.seat,NEW.ticket,NEW.event,NEW.note,
				jsonb_build_object('ops_event_id',NEW.id));
		ELSIF NEW.event='COS上行' OR (NEW.event='升件' AND NEW.seat='COS') THEN
			PERFORM ops_decision_write('seat',NEW.seat,NEW.ticket,'升件裁量',NEW.note,
				jsonb_build_object('ops_event_id',NEW.id,'src_event',NEW.event));
		END IF;
	EXCEPTION WHEN undefined_table OR undefined_function THEN NULL;
	END;
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS ops_decision_mirror_trg ON ops_event;
CREATE TRIGGER ops_decision_mirror_trg
	AFTER INSERT ON ops_event FOR EACH ROW
	EXECUTE FUNCTION ops_decision_mirror();

-- ③b lesson verdicted 镜像（裁定席 verdict 同写——两案表同函数）。
CREATE OR REPLACE FUNCTION ops_decision_lesson_verdict() RETURNS trigger AS $$
BEGIN
	BEGIN
		PERFORM ops_decision_write('seat','裁定席',TG_TABLE_NAME||'#'||NEW.id,'裁决',
			NEW.verdict_ref,
			jsonb_build_object('lesson_id',NEW.id,'table',TG_TABLE_NAME,'status',NEW.status));
	EXCEPTION WHEN undefined_table OR undefined_function THEN NULL; -- 部署窗回落（③a 同式）
	END;
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS ops_decision_econ_trg ON fin_econ_lesson;
CREATE TRIGGER ops_decision_econ_trg
	AFTER UPDATE OF status,verdict_ref ON fin_econ_lesson FOR EACH ROW
	WHEN (NEW.status='verdicted' AND OLD.status IS DISTINCT FROM 'verdicted')
	EXECUTE FUNCTION ops_decision_lesson_verdict();

DROP TRIGGER IF EXISTS ops_decision_eng_trg ON fin_eng_lesson;
CREATE TRIGGER ops_decision_eng_trg
	AFTER UPDATE OF status,verdict_ref ON fin_eng_lesson FOR EACH ROW
	WHEN (NEW.status='verdicted' AND OLD.status IS DISTINCT FROM 'verdicted')
	EXECUTE FUNCTION ops_decision_lesson_verdict();

-- ④ 授权（镜像触发器 INVOKER——事件行写角色须持 ops_decision 写权）。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='research_rw') THEN
		EXECUTE 'GRANT INSERT,SELECT ON ops_decision TO research_rw';
		EXECUTE 'GRANT USAGE ON SEQUENCE ops_decision_id_seq TO research_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='review_rw') THEN
		EXECUTE 'GRANT INSERT,SELECT ON ops_decision TO review_rw';
		EXECUTE 'GRANT USAGE ON SEQUENCE ops_decision_id_seq TO review_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT INSERT,SELECT ON ops_decision TO roster_rw';
		EXECUTE 'GRANT USAGE ON SEQUENCE ops_decision_id_seq TO roster_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
		EXECUTE 'GRANT SELECT ON ops_decision TO o_fin_ro';
	END IF;
END $$;
