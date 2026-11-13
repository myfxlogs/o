-- 0172 退役验尸案类（业主令 2026-11-13「经验管接通」件①）。
-- 病灶：策略退役=最大经验蒸发口——fin_strategy_transition 2175 退役全留痕
--   （from_state+reason）而 fin_econ_lesson 策略级死法零产线（26 案全仓位级
--   trade_postmortem/rootcause_diag）；举一反三器官有矿无料——死两千余次
--   无一条策略级判据可归纳。
-- 本迁=kind 集扩 'retire_postmortem'：
--   entity_ref=strategy#<id>（UNIQUE(kind,entity_ref) 一策略一验尸案幂等——
--   产线扫 transition 行未配套者补产，ON CONFLICT DO NOTHING 天然幂等）；
--   numbers 载策略档+死法+一生账（template/inst/venue/direction/notional/
--   from_state/reason/age_days/positions/realized_usd/paper_net_usd/execs/
--   proposal_ref——裁定席归纳素材全机读，不虚构 venue 外事实）。
--   产线=internal/portfolio/lifecycle/retireminer.go（scan 尾段慢产——
--   pending<20 自节流对齐裁定消化节奏，35d 窗喂新死优先）。
-- 约束换名纪律：同 fin_0134——定位含 rootcause_diag 字面量的 kind CHECK
--   真名 DROP 再 ADD 具名（幂等：定位不到=已新集）。

DO $$
DECLARE c text;
BEGIN
	SELECT conname INTO c FROM pg_constraint
		WHERE conrelid='fin_econ_lesson'::regclass AND contype='c'
			AND pg_get_constraintdef(oid) LIKE '%rootcause_diag%';
	IF c IS NOT NULL THEN
		EXECUTE format('ALTER TABLE fin_econ_lesson DROP CONSTRAINT %I', c);
	END IF;
END $$;
DO $$ BEGIN
	ALTER TABLE fin_econ_lesson ADD CONSTRAINT fin_econ_lesson_kind_chk
		CHECK (kind IN ('falsified_verdict','settled_loss','cost_diverge',
			'venue_degrade','flip_exit','semantic_caliber','trade_postmortem',
			'rootcause_diag','retire_postmortem'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
COMMENT ON COLUMN fin_econ_lesson.kind IS '案类闭集——retire_postmortem=策略退役验尸案（fin_0172：lifecycle 矿工扫 transition 未配套行产 pending，entity_ref=strategy#<id>，numbers=策略档+死法+一生账——2175 死零教训蒸发口根修）';
