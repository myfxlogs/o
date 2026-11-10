-- 0161 策略退役窄门函数——fin_retire(id,reason) SECURITY DEFINER。
-- 背景（业主令 2026-11-10「策略上退役=系统自裁」+首场实证坑）：
-- stratretire 微件以 roster_rw 跑 lifecycle.Transition 被拒——
-- fin_strategy UPDATE 仅属主 o 持有（授权面实核）,roster_rw 只 SELECT。
-- 宽授 UPDATE=会话可绕 Transition 裸写钱径表（审计留痕径破）;
-- 正解=o 属主 SECURITY DEFINER 函数:EXECUTE 授权即最窄写面,
-- 函数体内做 Transition 同构两件（合法迁校验+state 翻转+
-- transition 留痕,同事务）——退役对全态合法（闭集唯一例外=已退役
-- 重复迁,函数显式拒）,Go 侧 legal 图无漂移面。
-- ops_decision 墓志铭件不复制（decision.WriteEsc 是 Go 装配——
-- 函数只担钱径正账,决策记忆缺口诚实注此:自动退役/手动退役
-- 审计面差一行记忆行,后续桥补;transition 行=权威留痕）。
CREATE OR REPLACE FUNCTION fin_retire(p_id bigint, p_reason text)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
DECLARE v_from text;
BEGIN
	IF p_reason IS NULL OR btrim(p_reason) = '' THEN
		RAISE EXCEPTION '退役判词空——判词必填（留痕域）';
	END IF;
	SELECT state INTO v_from FROM fin_strategy WHERE id = p_id FOR UPDATE;
	IF NOT FOUND THEN RAISE EXCEPTION '策略#% 不存在', p_id; END IF;
	IF v_from = 'retired' THEN
		RAISE EXCEPTION '策略#% 已退役（重复迁拒）', p_id;
	END IF;
	UPDATE fin_strategy SET state='retired', entered_at=now() WHERE id=p_id;
	INSERT INTO fin_strategy_transition(strategy_id,from_state,to_state,reason)
		VALUES(p_id, v_from, 'retired', p_reason);
END $$;
REVOKE ALL ON FUNCTION fin_retire(bigint,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION fin_retire(bigint,text) TO roster_rw;
