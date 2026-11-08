-- 0147 T-0489 径A：simvenue 仓写径分域——仓行落 sim_ 命名域表，fin_* 正本只纳真户。
--
-- 病灶：simdrill 矩阵/演习产跑的 simvenue 仓直写 fin_position/fin_execution 正本
-- （fin_position #35-39 实证——业主 OKX 对账扑空、postmortem/D1 对 sim 立案空烧
-- 诊断额度、T-0145 残仓判据腐蚀）。径 A=结构性分域：写域由执行件显式声明
-- （live=fin_*|sim=sim_*），正本表查询面零改动天然隔离。
--
-- 五表镜像（仓生命周期全簇——execution/position/order_leg/cashflow/guard_close
-- 引用闭合同域内，卫星表留 fin 域则 execution_id/position_ref 撞号回染真户行）：
--
--   sim_execution / sim_position / sim_order_leg / sim_cashflow / sim_guard_close
--
-- LIKE INCLUDING ALL：列+DEFAULT+CHECK+索引随正本现行形复刻（venue_err_*/sz_unit/
-- settled_*/bare_bag_*/trigger_verdict/kind 闭集全随）；id DEFAULT nextval 逐字
-- 复刻=sim 与 fin 同序列取号——仓/执行 id 全域唯一不撞（下游各自选读时行 id
-- 无域歧义；序列被 sim 消耗产 fin 空号=serial 本非无缝语义，无害）。
--
-- 触发器不复刻（LIKE 本不携）：ev_position/ev_cashflow NOTIFY 与
-- fin_active_set_maintain 对 sim 域天然不燃——演习拍由 harness 显式 ScanOnce
-- 驱动，真户事件脊零唤醒零污染；fin_cashflow_class_stamp 分类戳同构补挂
-- sim_cashflow（kind→class 口径两域同形）。
-- FK 不随 LIKE：order_leg/position 外键显式指 sim_execution（域内闭合同）。

CREATE TABLE IF NOT EXISTS sim_execution (LIKE fin_execution INCLUDING ALL);
COMMENT ON TABLE sim_execution IS 'sim 域执行账（T-0489 径A）——simvenue/simdrill 产跑仓行写域；fin_execution 只纳真户，id 与 fin_execution 同序列全域唯一';

CREATE TABLE IF NOT EXISTS sim_position (LIKE fin_position INCLUDING ALL);
COMMENT ON TABLE sim_position IS 'sim 域持仓账（T-0489 径A）——postmortem/D1/仓巡/经营事件正本面对其天然不入';
ALTER TABLE sim_position
    ADD FOREIGN KEY (execution_id) REFERENCES sim_execution(id);

CREATE TABLE IF NOT EXISTS sim_order_leg (LIKE fin_order_leg INCLUDING ALL);
COMMENT ON TABLE sim_order_leg IS 'sim 域腿账（T-0489 径A）——execution_id 闭合同 sim_execution';
ALTER TABLE sim_order_leg
    ADD FOREIGN KEY (execution_id) REFERENCES sim_execution(id);

CREATE TABLE IF NOT EXISTS sim_cashflow (LIKE fin_cashflow INCLUDING ALL);
COMMENT ON TABLE sim_cashflow IS 'sim 域现金流账（T-0489 径A）——腿级费/结算行；position_ref 闭合同 sim_position';
CREATE TRIGGER fin_cashflow_class_stamp BEFORE INSERT OR UPDATE OF kind ON sim_cashflow
    FOR EACH ROW EXECUTE FUNCTION fin_cashflow_class_stamp();

CREATE TABLE IF NOT EXISTS sim_guard_close (LIKE fin_guard_close INCLUDING ALL);
COMMENT ON TABLE sim_guard_close IS 'sim 域守卫平仓账（T-0489 径A）——position_id 闭合同 sim_position';
