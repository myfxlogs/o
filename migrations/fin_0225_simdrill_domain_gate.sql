-- fin_0225: simdrill 演习域写径闸（浑水式法证 2026-11-21 劈③整改）。
-- 病灶:演习现金行(venue='sim')与真钱同簿共表,防泄全靠消费面
-- 「记得写 venue<>'sim' 谓词」的约定——写径零闸,夹具行写进
-- 真仓 position_ref 也照收(隔离债工单#20819 同坑)。
-- 本件升库层前置闸(同 sem_gate/0223 哲学):venue='sim' 行必须
-- 可溯演习域——双径认亲:
--   径① 已挂仓(position_ref 非空):仓→fin_strategy.template_ref
--        LIKE 'simdrill%'(演习 ephemeral 策略全谱打此前缀,
--        drill.go:40 StratTemplate 钦定)。
--   径② 未挂仓(position_ref NULL——开仓径腿行先于仓生,
--        linkLegCash 事后补挂):source_table='fin_order_leg' 且
--        source_ref→fin_order_leg→fin_execution→fin_strategy
--        同样须 simdrill%。
-- UPDATE 段同闸:sim 行 position_ref 补挂/改挂目标仓非演习域即拒
-- (防事后把演习行挂进真仓漂账)。存量 58 行不受影响(闸只管新写)。
CREATE OR REPLACE FUNCTION fin_cashflow_sim_gate() RETURNS trigger AS $$
BEGIN
    IF NEW.venue <> 'sim' THEN
        RETURN NEW;
    END IF;
    -- 径①：已挂仓——仓→策略演习域判
    IF NEW.position_ref IS NOT NULL AND EXISTS (
        SELECT 1 FROM fin_position p
        JOIN fin_strategy s ON s.id = p.strategy_ref
        WHERE p.id = NEW.position_ref AND s.template_ref LIKE 'simdrill%'
    ) THEN
        RETURN NEW;
    END IF;
    -- 径②：未挂仓腿行——源腿溯 execution→策略演习域判
    IF NEW.position_ref IS NULL AND NEW.source_table = 'fin_order_leg' AND EXISTS (
        SELECT 1 FROM fin_order_leg l
        JOIN fin_execution e ON e.id = l.execution_id
        JOIN fin_strategy s ON s.id = e.strategy_ref
        WHERE l.id = NEW.source_ref AND s.template_ref LIKE 'simdrill%'
    ) THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION 'fin_cashflow venue=sim 非演习域行——sim 域=仓/腿链溯 simdrill 策略专属,实账写径物理拒(隔离闸 fin_0225;position_ref=% source_table=%)', NEW.position_ref, NEW.source_table;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS fin_cashflow_sim_gate_trg ON fin_cashflow;
CREATE TRIGGER fin_cashflow_sim_gate_trg
    BEFORE INSERT OR UPDATE OF position_ref ON fin_cashflow
    FOR EACH ROW EXECUTE FUNCTION fin_cashflow_sim_gate();
