-- 0083 T-0216 返修②投影提案桥自修（SG 镜像侧；md 研究侧=md_1074 同构——
--   同一发布 boot 内双侧齐落，见 md_1074 头注）：
-- 病灶：提案桥落地面 direction 封闭词表（fin_0060）无 perp_quarter_spread——
--   md_1074 去分叉后镜像 uEval/uProposal 携新 direction 落 SG 即 CHECK 拒=
--   批毒丸重试环；且 SG 存量镜像行仍持旧 direction（uEval 冲突判自然键
--   (inst_id,template,direction) 不含 direction SET 列——不改此行则 md 回填后
--   同一 eval 在 SG 分裂双行）。桥自修=约束扩容+镜像行同构回填一次齐。
-- 附：fin_0054 去重闸函数无需改——(inst_id,direction) 键随词表去分叉自然归一。

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

-- 镜像行同构回填（先扩容后回填）：eval/proposal 按 template 锚定；
-- fin_strategy=SG 正本（md 侧只读副本由反向桥 diff 自愈，禁直改）。
UPDATE fin_opp_eval SET direction='perp_quarter_spread'
  WHERE template='perp_quarter_spread' AND direction='basis_quarter';
UPDATE fin_proposal p SET direction='perp_quarter_spread'
  FROM fin_opp_eval e WHERE e.id=p.eval_id
  AND e.template='perp_quarter_spread' AND p.direction='basis_quarter';
UPDATE fin_strategy SET direction='perp_quarter_spread'
  WHERE template_ref='perp_quarter_spread' AND direction='basis_quarter';
