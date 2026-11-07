-- 0133 T-0431 根因诊断席——根因案型入账（案卷驱动开放考古排班+判词归流）。
-- 病灶（业主判词）：机器会分类不会归因——0420 结案判定机只产「哪类异常」，
-- 「为什么异常」无座位（pos34 leg.sz 单位错配靠业主问+人肉考古 7h 定位）；
-- 工程径自循环断在诊断：detect→升件→案卷→【诊断缺】→判词→修法。
-- 本迁=fin_econ_lesson 两集扩：
--   ① kind 集扩 'rootcause_diag'——结案判定机异常件（残袋/认知集外触发/
--     守卫环 attempts≥2——账实差未定性三案卷源）自动生成根因诊断案
--     （entity_ref=position#<id>，与 trade_postmortem 同 ref 并行；
--     UNIQUE(kind,entity_ref) 一仓一诊断案幂等）；
--   ② status 集扩 'drafted'——诊断草案已落位待 DM 裁（草案=假设非判词
--     人机分界钉死：机器不自批）；水印闸只数 pending——drafted 不烧拍
--     （待裁面归 DM，verdicted 终态照旧）。
-- 约束换名纪律：原 CHECK 为内联无名式（自动名随落库序漂移）——按约束定义
--   子串定位真名 DROP 再 ADD 具名约束（部署窗幂等：定位不到=已新集，ADD
--   具名重放撞名即错显形非静默；fin_0131 同款）。

-- ① kind 集扩 rootcause_diag（定位含 trade_postmortem 字面量的 kind CHECK
--   真名——status 集无此词不撞）。
DO $$
DECLARE c text;
BEGIN
	SELECT conname INTO c FROM pg_constraint
		WHERE conrelid='fin_econ_lesson'::regclass AND contype='c'
			AND pg_get_constraintdef(oid) LIKE '%trade_postmortem%';
	IF c IS NOT NULL THEN
		EXECUTE format('ALTER TABLE fin_econ_lesson DROP CONSTRAINT %I', c);
	END IF;
END $$;
DO $$ BEGIN
	ALTER TABLE fin_econ_lesson ADD CONSTRAINT fin_econ_lesson_kind_chk
		CHECK (kind IN ('falsified_verdict','settled_loss','cost_diverge',
			'venue_degrade','flip_exit','semantic_caliber','trade_postmortem',
			'rootcause_diag'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
COMMENT ON COLUMN fin_econ_lesson.kind IS '案类闭集——rootcause_diag=根因诊断案（T-0431：结案判定机异常件自动生成诊断任务，entity_ref=position#<id>，numbers.case=案卷+seeds=假说族种子）';

-- ② status 集扩 drafted（定位含 verdicted 字面量的 status CHECK 真名）。
DO $$
DECLARE c text;
BEGIN
	SELECT conname INTO c FROM pg_constraint
		WHERE conrelid='fin_econ_lesson'::regclass AND contype='c'
			AND pg_get_constraintdef(oid) LIKE '%verdicted%';
	IF c IS NOT NULL THEN
		EXECUTE format('ALTER TABLE fin_econ_lesson DROP CONSTRAINT %I', c);
	END IF;
END $$;
DO $$ BEGIN
	ALTER TABLE fin_econ_lesson ADD CONSTRAINT fin_econ_lesson_status_chk
		CHECK (status IN ('pending','drafted','verdicted','canon'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
COMMENT ON COLUMN fin_econ_lesson.status IS 'pending=待裁/待诊断；drafted=诊断草案落位待 DM 裁（T-0431 人机分界——草案非判词机器不自批）；verdicted=裁毕；canon=成例入库';
