-- fin_0186: 器官镜组探针读面授权（业主令 2026-11-15⑦「跟自然人一样的能力」批造——
--   对内盲区外照层四探：产线静默/前提疑死/判词失真/器官自审）。
-- 病灶：dispatch 器官探（roster_rw）对 fin_opp_eval/fin_strategy 零读权——
--   「启用廊→产出流」探物理不可行（产出表读不出，盲面恒盲）。
-- 口径：补授 SELECT（读面非写面，与 fin_0173/0182 同形——角色存在才授幂等）。
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'roster_rw') THEN
		GRANT SELECT ON fin_opp_eval TO roster_rw;
		GRANT SELECT ON fin_strategy TO roster_rw;
		GRANT SELECT ON fin_opp_hypothesis TO roster_rw;
	END IF;
END $$;
