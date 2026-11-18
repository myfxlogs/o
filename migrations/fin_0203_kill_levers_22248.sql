-- fin_0203_kill_levers_22248.sql —— 工单#22248 件②③（件①停摆键=码面
-- earn.residual_clear_halted，同 commit internal/earn）：
--
--   ②事故态可读:fin_guard_close_status 窄视图 + roster_rw SELECT。
--     roster_rw 拒读 fin_guard_close=自我席事故态只凭票面文字猜裸空
--     （仓#86 裸空 4h30m 实证——dispatch_self 跟进钟只能用守卫环升件
--     近窗代理非精确态）。窄视图载 position→inst 上下文+state/
--     attempts/last_error+裸袋三列；roster_rw 窄白名单授权径同形
--     fin_0103/fin_0111（视图 owner 读底表，grant 面只盖视图）。
--
--   ③到人径通闸:sem_matrix 补「条件请求×自我」——self_runbook 令
--     自我席写条件请求问董事会而矩阵无此组合=物理拒写（闸做对了，
--     是契约令写了禁词——VVV 事故时按铃径不存在实证）。只放行事件
--     词，写面频度仍由自我席契约束（条件请求=到业主径，每用必载实据）。

BEGIN;

CREATE OR REPLACE VIEW fin_guard_close_status AS
SELECT g.position_id, p.execution_id, p.inst_id,
       g.state, g.trigger_kind, g.attempts,
       g.last_try_at, g.last_attempt_at, g.closed_at, g.updated_at,
       g.last_error,
       p.bare_bag_ccy, p.bare_bag_qty, p.bare_bag_usd
FROM fin_guard_close g
JOIN fin_position p ON p.id = g.position_id;

DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
        EXECUTE 'GRANT SELECT ON fin_guard_close_status TO roster_rw';
    END IF;
END $$;

-- 补链（同 fin_0166 存量对迁外入册先例）：('条件请求','处置席') 行
-- 生产在册而迁移链内无种子=链漂移——闸按 event 名激活,本迁把
-- 事件名首次装入射程会致测试库 处置席 行缺席而拒写存量径。
INSERT INTO ops_event_sem_matrix(event,seat,note_like,basis) VALUES
('条件请求','处置席','','纪元3会话到业主径写件（存量对补链——生产在册链内无种子,fin_0203 迁外入册使测试链与生产同形）'),
('条件请求','自我','','自我席到人径——工单#22248③：self_runbook 令自我席写条件请求问董事会而矩阵无此组合=物理拒写（VVV 事故按铃径不存在实证）。放行事件词不放宽频度——条件请求=到业主径,每用必载实据非问句装饰')
ON CONFLICT (event,seat) DO NOTHING;

COMMIT;
