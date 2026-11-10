-- 0159 审计臂落账——ops_event_sem_matrix 补「唤起×审计机」放行组合。
-- 背景（D-0046 实账）：审计臂日拍唤审计会话,跨机单飞锁与唤知机同构
-- ——ops_event seat='审计机' event='唤起' 行在案=30min 在飞判据。
-- 「唤起」在闸内词表而审计机无矩阵行→ops_event_sem_gate loud 拒写
-- （2026-11-10 首场手跑实证 RAISE EXCEPTION——臂首拍即哑,闸做对了:
-- 新 seat 未经裁处入册本就该拒）。
-- 本迁只补放行一行;审计臂其余写面（升件/审计/审计疑点/审计复证/
-- 唤起败）皆闸外词直通不涉——白名单收紧面不松。
INSERT INTO ops_event_sem_matrix(event,seat,note_like,basis) VALUES
('唤起','审计机','','审计臂 D-0046——跨机单飞锁 seat 分账（唤知环=唤知机/审计臂=审计机；锁键 7970 与 7969 分域,同构语义同座不同账）')
ON CONFLICT (event,seat) DO NOTHING;
