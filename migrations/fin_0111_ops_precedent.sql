-- fin_0111 T-0334 判例索引面——升件裁转机器判例，同类自动裁（判例法化）。
-- 三层确定性面：
--   ① 判例登记面 ops_precedent——载体选型钦定=专用表：
--      · ops_event 结构化行=note 自由文无列可载 class/scope/ruling，
--        精确匹配须解串=违「只许结构化精确匹配」；
--      · docs/decisions 机读索引=决策席章面，机查须 git 往返非运行时
--        面，且判例=运行态数据非方向档；
--      · 专用表=列级等值匹配+触发器同拍登记+闸门库内零延迟（上行闸前
--        =升件插行即判，先于一切打包扫描）。
--   ② 匹配闸=ops_event AFTER INSERT 行触发器 ops_precedent_trg——
--      升件带类行（esc_class/esc_scope 非空）落库即查判例：同类同域
--      精确等值命中→ops_event '先例适用' 行（升件消化，打包面见行即略）；
--      未命中/出界→零动作照常上行（无命中不猜）。
--   ③ 透明径='先例适用' 行 note 携 升件#id 判例#id 源裁#id——业主回流
--      可见可翻案；翻案=同 class+scope 新判例登记（supersedes 链）。
-- 登记面同嵌触发器：裁决/业主批 落行同拍→同票最新未消化带类升件判例化
-- （「升级」裁词不登记——升级=再上行非消化，判例化反吞升件）。
-- 红线类级黑名单结构性排除：class 或 scope 命中 {资金扩投,新平台,
-- 席位立换,红线} 即不裁不登记——即使存在貌似同类判例永不自动裁。
-- 判类册=scripts/meta/precedent_classes.tsv（类词登记+红线标——写者
-- 选类参照；匹配为等值非模糊，册不参判）。带类升件写径=
-- scripts/meta/escalate.sh。
-- 故障语义：触发器两臂皆 EXCEPTION 兜底 fail-open——判例面异常仅
-- WARNING，升件/裁行落库永不因判例面回滚（吞判例不吞流水）。

ALTER TABLE ops_event ADD COLUMN IF NOT EXISTS esc_class text;
ALTER TABLE ops_event ADD COLUMN IF NOT EXISTS esc_scope text;
COMMENT ON COLUMN ops_event.esc_class IS '升件类（T-0334 判例面匹配键——判类册登记词，NULL=不入判例面照常上行）';
COMMENT ON COLUMN ops_event.esc_scope IS '升件适用域界桩（T-0334 匹配键之二——等值精确，越界=未命中）';

CREATE TABLE IF NOT EXISTS ops_precedent (
	id          bigserial PRIMARY KEY,
	ts          timestamptz NOT NULL DEFAULT now(),
	class       text NOT NULL,
	scope       text NOT NULL,
	ruling      text NOT NULL,
	source_row  bigint NOT NULL REFERENCES ops_event(id),
	source_esc  bigint REFERENCES ops_event(id),
	active      boolean NOT NULL DEFAULT true,
	supersedes  bigint REFERENCES ops_precedent(id)
);
COMMENT ON TABLE ops_precedent IS '升件判例簿（T-0334——同类同域升件自动裁判据；append-only 语义，翻案=新行替活 supersedes 链）';
COMMENT ON COLUMN ops_precedent.source_row IS '源裁行 id——判例出生证（ops_event 裁决/业主批 行）';
COMMENT ON COLUMN ops_precedent.source_esc IS '被裁升件行 id（登记触发器回填——判例出处双向可查）';
-- 同类同域活判例唯一（翻案=旧行落 active=false+新行——部分唯一索引钉死）
CREATE UNIQUE INDEX IF NOT EXISTS ops_precedent_live
	ON ops_precedent(class,scope) WHERE active;
CREATE INDEX IF NOT EXISTS ops_precedent_source ON ops_precedent(source_row);

CREATE OR REPLACE FUNCTION ops_precedent_trg() RETURNS trigger AS $$
DECLARE
	p record;
	e record;
	oldid bigint;
BEGIN
	-- 闸臂：升件带类行→同类同域等值匹配；红线类/域结构性不裁。
	IF NEW.event='升件' AND NEW.esc_class IS NOT NULL THEN
		BEGIN
			IF NOT (NEW.esc_class = ANY(ARRAY['资金扩投','新平台','席位立换','红线'])
			     OR NEW.esc_scope = ANY(ARRAY['资金扩投','新平台','席位立换','红线'])) THEN
				SELECT id,ruling,source_row INTO p FROM ops_precedent
				 WHERE class=NEW.esc_class AND scope=NEW.esc_scope AND active
				 ORDER BY id DESC LIMIT 1;
				IF FOUND THEN
					INSERT INTO ops_event(seat,event,ticket,note)
						VALUES('判例闸','先例适用',NEW.ticket,
						       '升件#'||NEW.id||' 判例#'||p.id||' 源裁#'||p.source_row
						       ||'——'||p.ruling);
				END IF;
			END IF;
		EXCEPTION WHEN OTHERS THEN
			RAISE WARNING 'ops_precedent 闸臂 fail-open: %', SQLERRM;
		END;
	END IF;
	-- 登记臂：裁决/业主批落行→同票最新未消化带类升件判例化。
	-- 「升级」裁词不登记（升级=再上行非消化）；红线类不登记（双闸）。
	-- 去重键=升件后无裁决/业主批跟行；先例适用行不排除——被闸臂自动
	-- 消化的升件再落裁词=翻案，钦定翻案本身即新判例登记（supersedes）。
	IF NEW.event IN ('裁决','业主批') AND NEW.ticket<>''
	   AND NEW.note NOT LIKE '升级%' THEN
		BEGIN
			SELECT oe.id AS esc_id, oe.esc_class AS ec, oe.esc_scope AS es INTO e
			  FROM ops_event oe
			 WHERE oe.ticket=NEW.ticket AND oe.event='升件'
			   AND oe.esc_class IS NOT NULL AND oe.id<NEW.id
			   AND NOT EXISTS (SELECT 1 FROM ops_event r
			                   WHERE r.ticket=oe.ticket AND r.event IN ('裁决','业主批')
			                     AND r.id>oe.id AND r.id<NEW.id)
			 ORDER BY oe.id DESC LIMIT 1;
			IF FOUND AND NOT (e.ec = ANY(ARRAY['资金扩投','新平台','席位立换','红线'])
			               OR e.es = ANY(ARRAY['资金扩投','新平台','席位立换','红线'])) THEN
				UPDATE ops_precedent SET active=false
				 WHERE class=e.ec AND scope=e.es AND active
				RETURNING id INTO oldid;
				INSERT INTO ops_precedent(class,scope,ruling,source_row,source_esc,supersedes)
					VALUES(e.ec,e.es,NEW.note,NEW.id,e.esc_id,oldid);
			END IF;
		EXCEPTION WHEN OTHERS THEN
			RAISE WARNING 'ops_precedent 登记臂 fail-open: %', SQLERRM;
		END;
	END IF;
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER ops_precedent_trg AFTER INSERT ON ops_event
	FOR EACH ROW EXECUTE FUNCTION ops_precedent_trg();

-- 写/读面授权（条件式——角色缺席不炸，全新部署哑死防）：
--   触发器=INVOKER 语义，升件/裁行写者须持 ops_precedent 读写权
--   （roster_rw=席位自报主凭据/research_rw=fin_0086 自报同源）；
--   o_fin_ro 窄读面补 SELECT（COS 收件箱判例可视）。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT SELECT,INSERT,UPDATE ON ops_precedent TO roster_rw';
		EXECUTE 'GRANT USAGE ON SEQUENCE ops_precedent_id_seq TO roster_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='research_rw') THEN
		EXECUTE 'GRANT SELECT,INSERT,UPDATE ON ops_precedent TO research_rw';
		EXECUTE 'GRANT USAGE ON SEQUENCE ops_precedent_id_seq TO research_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
		EXECUTE 'GRANT SELECT ON ops_precedent TO o_fin_ro';
	END IF;
END $$;

-- 首三条今日真判例回填（验收锚②——source_row=ops_event 真裁行，
-- 源行实存在才落：迁移面回放于无源行实例=零行不炸）：
--   ① 部署窗口自裁委托——业主批1652「业主令直派部署窗——O-Dev施工」
--     （业主令直裁无源升件行——source_esc 空）；
--   ② o_fin_ro 补授先例——升件1564 裁决1569「升件1564补裁采纳A落返修①」；
--   ③ 甲向采纳类——升件1593 裁决1605「返修①五fin表升bridge-reverse」=采纳修法候选①。
INSERT INTO ops_precedent(class,scope,ruling,source_row)
	SELECT '部署窗','O-Dev','直派O-Dev施工——部署窗事项业主已委托不再上行（业主批1652）',1652
	WHERE EXISTS (SELECT 1 FROM ops_event WHERE id=1652 AND event='业主批')
	  AND NOT EXISTS (SELECT 1 FROM ops_precedent WHERE class='部署窗' AND scope='O-Dev' AND active);
INSERT INTO ops_precedent(class,scope,ruling,source_row,source_esc)
	SELECT '授权补授','o_fin_ro','白名单迁移补授——SELECT-only 窄读面 GRANT 随迁移落（升件1564裁1569采纳A；fin_0108/0109 先例同径）',1569,1564
	WHERE EXISTS (SELECT 1 FROM ops_event WHERE id=1569 AND event='裁决')
	  AND NOT EXISTS (SELECT 1 FROM ops_precedent WHERE class='授权补授' AND scope='o_fin_ro' AND active);
INSERT INTO ops_precedent(class,scope,ruling,source_row,source_esc)
	SELECT '选项采纳','修法候选','采纳建议甲向——升件列修法候选施工注建议，裁采纳甲落返修（升件1593裁1605采纳候选①）',1605,1593
	WHERE EXISTS (SELECT 1 FROM ops_event WHERE id=1605 AND event='裁决')
	  AND NOT EXISTS (SELECT 1 FROM ops_precedent WHERE class='选项采纳' AND scope='修法候选' AND active);
