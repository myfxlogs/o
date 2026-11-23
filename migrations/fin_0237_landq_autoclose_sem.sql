-- fin_0237_landq_autoclose_sem.sql —— 落成自动销票销行合法化（工单O-33705）。
--
-- 病灶（2026-11-23 实证）：land_queue 落成尾销票细胞（O-33182 件）
-- 逐票 INSERT seat='落地件' event='处置' note='工单#<tid> 销——落成
-- 自动销 支=<br> merge=<sha>'——ops_event_sem_gate 合法性矩阵缺
-- (处置,落地件) 对→BEFORE INSERT trigger RAISE 拒写；脚本侧异常被
-- >/dev/null 2>&1+say 吞，而 DID 在 INSERT 前已追加→「落成自动销
-- 支=… 票=<csv>」假汇总行照写。事实账称销未销六行在案：
-- ops_event#33461 #33592 #33594 #34342 #34502 #34517；全库
-- seat='落地件'×event='处置' 实测零行=机制从未成功过一次。
--
-- 本迁只补放行一行，note_like 收窄到自动销形——落地件其余处置面
-- 不放行（销票语义不变：本行=机制销行非人工销，act 视图 close
-- 派生同形——工单#<rel>[^—#]{0,20}销[—：:，,] 谓词实测命中）。
INSERT INTO ops_event_sem_matrix(event,seat,note_like,basis) VALUES
('处置','落地件','工单#% 销——落成自动销%','落地件自动销票机制行（O-33705——land_queue 落成尾销票细胞；note 形收窄=工单#<tid> 销——落成自动销 支=<br> merge=<sha>，人工销/其余处置面不放行）')
ON CONFLICT (event,seat) DO NOTHING;
