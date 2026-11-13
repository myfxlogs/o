-- fin_0173: md_klines 反向桥源表研究面读授（工单#13065 切片④附）。
-- 病灶：domain 翻转 bridge→bridge-reverse 后桥审断言 reverse 源表
--   research_rw.SELECT——fin_0171 只授 projector_rw（正向谱下投影读件），
--   研究面按需读 SG 正本口径缺授即红。
-- 口径：补授 SELECT（读面非写面）——角色存在才授，与 fin_0170 同形。
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'research_rw') THEN
		GRANT SELECT ON md_klines TO research_rw;
	END IF;
END $$;
