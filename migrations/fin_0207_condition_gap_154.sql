-- fin_0207 —— 工单#23077②:fin_condition_gap id=154 纪元2 verdict 请托残件定性销
--
-- 行=2026-11-08 T-0461「13张策略上膛」executor verdict 请托,state=pending 超 TTL
-- 9d+。消费面=旧 condreq 人工裁道——纪元2 官僚层退役后无人读此表 pending 行
-- (armed→open 判归 lifecycle 机械闸)。FIN/RO 角色皆无写权,销件只能走迁
-- 移根径(owner 批 UPDATE)。定性销非删除——行留档,state 翻 dismissed
-- (CHECK 闭集=pending/fulfilled/rejected/dismissed,'resolved' 出集撞闸——
-- 升件#24758 boot crashloop 实证)。
UPDATE fin_condition_gap
   SET state='dismissed', resolved_at=now(),
       resolve_note='设计态过期:condreq道废armed归lifecycle机械闸——纪元2 T-0461 verdict请托「13张策略上膛」消费面退役,行超TTL9d,定性销(工单#23077②)'
 WHERE id=154 AND state='pending';
