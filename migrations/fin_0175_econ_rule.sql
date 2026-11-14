-- fin_0175: fin_econ_rule——canon 可执行化（业主令 2026-11-14「经验→知识→闸」
--   件②,对拍 Voyager/SkillSmith 技能语义:可机读教训不落散文落规约行——
--   admit_block 由生命周期 admitProposals 机械执行,「lethal 簇禁再喂」
--   从契约劝会话变机器闸,裁定席 INSERT 即生效,active=false 撤规）。
-- match 键闭集 {template_ref,inst_id,direction} AND 匹配;未知键/空 match=
-- 规约不适用(fail-closed 不拦);kind 闭集现只 admit_block(早退/仓位闸
-- 待判据沉淀再扩——扩 kind 须同迁 CHECK)。
CREATE TABLE IF NOT EXISTS fin_econ_rule (
    id         bigserial PRIMARY KEY,
    kind       text NOT NULL CHECK (kind='admit_block'),
    match      jsonb NOT NULL,
    source     text NOT NULL,
    note       text NOT NULL DEFAULT '',
    active     boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(kind,match)
);
COMMENT ON TABLE fin_econ_rule IS
  '经济规约册(fin_0175)——裁定席把可机读 canon 落成行,生命周期 admitProposals
   机械执行;kind=admit_block:match{template_ref?inst_id?direction?}AND 匹配
   的提案不建策略留 fin_proposed;source=canon:<条目id>∨lesson#<id> 锚出处;
   active=false=撤规不删行(判词可回查)';
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'roster_rw') THEN
        EXECUTE 'GRANT SELECT,INSERT,UPDATE ON fin_econ_rule TO roster_rw';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'o_fin_ro') THEN
        EXECUTE 'GRANT SELECT ON fin_econ_rule TO o_fin_ro';
    END IF;
END $$;
