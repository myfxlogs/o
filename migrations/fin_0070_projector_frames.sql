-- 0070 projector 回写帧（T-0168/D-0030 §四——SG 侧 fin_proposal 回写源）：
-- state 列组（state/retire_reason/retired_at——D-0030 写权矩阵）lifecycle 改态
-- 即发 ev_proposal（projector 反向听此回写 PG-md）。UPDATE OF state 按 SET 列
-- 清单触发（非值变触发，D-0032 §三-6 实证口径）——前向投影不写 state 列组=
-- 不自触发（回环禁断言在 projector 桩锚）。与 md 链 md_1067 正向帧同名异库。
CREATE OR REPLACE TRIGGER ev_proposal_sg_trg AFTER UPDATE OF state ON fin_proposal
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_proposal');
