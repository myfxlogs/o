-- 1074 T-0216 返修③提案去重语叉回填（md 研究侧；SG 镜像侧=fin_0083 同构——
--   同一发布 boot 内双侧齐落：projector uEval/uProposal 对新 direction 的
--   毒丸窗口与 natural-key 双行分叉均由「两侧迁移先于服务起」封死）：
-- 病灶：quarter_basis_long 与 perp_quarter_spread 共用 direction='basis_quarter'
--   且评估同 inst（季度期货键）——md_1054 去重闸键 (inst_id,direction) 双族
--   撞车，第二族提案 BEFORE INSERT RETURN NULL 静默吞（grammar run.go 预查
--   inst 单键同病更宽）。同 inst 跨族 direction 复用=去重语义分叉根。
-- 修法：direction 词表去分叉——perp_quarter_spread 族自立 direction（同模板
--   名，dcd_call/gap_carry 同构式）；存量行按 template 回填；闸函数无需改
--   （(inst_id,direction) 键在词表去分叉后语义自然归一）。

ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_direction_check;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread'));
ALTER TABLE fin_proposal DROP CONSTRAINT IF EXISTS fin_proposal_direction_check;
ALTER TABLE fin_proposal ADD CONSTRAINT fin_proposal_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge',
                       'basis_quarter','basis_quarter_short','lending_spread',
                       'earn_term_spread','vol_sell',
                       'long_perp_short_cfd','short_perp_long_cfd',
                       'perp_quarter_spread'));

-- 存量回填（先扩容后回填——新值过新闸；按 template 锚定防误伤他族同 direction 行）。
UPDATE fin_opp_eval SET direction='perp_quarter_spread'
  WHERE template='perp_quarter_spread' AND direction='basis_quarter';
UPDATE fin_proposal p SET direction='perp_quarter_spread'
  FROM fin_opp_eval e WHERE e.id=p.eval_id
  AND e.template='perp_quarter_spread' AND p.direction='basis_quarter';
