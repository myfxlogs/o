-- 1079 T-0299 fin_opp_hypothesis.state 扩 'parked'（D-0043 收敛）——
-- 在册 pending 卡冻结留 T-0300 做 venue 新族对照料：不裁不驳，解冻=
-- 改回 proposed。禁 UPDATE 撞 CHECK——先扩约束再迁移存量。
ALTER TABLE fin_opp_hypothesis DROP CONSTRAINT fin_opp_hypothesis_state_check;
ALTER TABLE fin_opp_hypothesis ADD CONSTRAINT fin_opp_hypothesis_state_check
  CHECK (state IN ('proposed','adopted','dismissed','parked'));
UPDATE fin_opp_hypothesis SET state='parked' WHERE state='proposed';
