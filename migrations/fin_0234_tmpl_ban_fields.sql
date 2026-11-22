-- fin_0234: fin_opp_template 禁因机读化三件套+名册视图 SG 镜像侧
-- （工单O-33393——md_1103 正本同形镜像：fin_opp_template 正本归 PG-md，
-- 本链=ghost 桥镜像表，projector venue_cand 列携 md→sg；DDL 归一化同形
-- 断言钉——列/约束/视图语句与 md_1103 逐字同）。回填 UPDATE 与 md 链
-- 同值——投影未至窗内镜像自洽，投影至后以 MD 正本为准覆写。
ALTER TABLE fin_opp_template ADD COLUMN IF NOT EXISTS disabled_reason text;
ALTER TABLE fin_opp_template ADD COLUMN IF NOT EXISTS ban_premise text;
ALTER TABLE fin_opp_template ADD COLUMN IF NOT EXISTS ban_premise_kind text;
ALTER TABLE fin_opp_template ADD COLUMN IF NOT EXISTS banned_at timestamptz;
ALTER TABLE fin_opp_template ADD COLUMN IF NOT EXISTS ban_last_verdict text;
ALTER TABLE fin_opp_template ADD COLUMN IF NOT EXISTS ban_last_check_at timestamptz;
COMMENT ON COLUMN fin_opp_template.disabled_reason IS '禁因人读——enabled=f 必填显式值（含「禁因失考」）；复审解禁后留痕历史';
COMMENT ON COLUMN fin_opp_template.ban_premise IS '禁用前提=可证伪命题——复核拍重核对象；NULL=无前提可核（candidate/测试面/失考）';
COMMENT ON COLUMN fin_opp_template.ban_premise_kind IS '前提类别机读键：venue_capability=venue面缺类/apr_regime=收益恒拒/owner_order=业主令（不机核）/none=非禁用';
COMMENT ON COLUMN fin_opp_template.banned_at IS '禁日——龄报锚；非禁用行 NULL';
COMMENT ON COLUMN fin_opp_template.ban_last_verdict IS '复核拍末次结论（falsified/upheld/unknown/owner_order——名册「前提成不成立」同答键）';
COMMENT ON COLUMN fin_opp_template.ban_last_check_at IS '复核拍末次核时——龄报锚之二';

ALTER TABLE fin_opp_template DROP CONSTRAINT IF EXISTS fin_opp_template_state_check;
ALTER TABLE fin_opp_template ADD CONSTRAINT fin_opp_template_state_check
  CHECK (state IN ('enabled','candidate','disabled','retired')) NOT VALID;

ALTER TABLE fin_opp_template DROP CONSTRAINT IF EXISTS fin_opp_template_ban_reason_check;
ALTER TABLE fin_opp_template ADD CONSTRAINT fin_opp_template_ban_reason_check
  CHECK (state <> 'disabled' OR disabled_reason IS NOT NULL) NOT VALID;

UPDATE fin_opp_template SET
  disabled_reason='钉定挂起回禁用——dcd族臂连败自停+业主驳回重启（ops_event#13201）；底判=MANAGE-151 venue物理缺类定案（canon:dcd-exit-venue-capability-immature，锚 ops_event#139/#141）',
  ban_premise='okx DCD 产品面（/api/v5/finance/sfp/dcd）venue 能力缺类/不可用——产品目录与订单面不供给',
  ban_premise_kind='venue_capability',
  banned_at='2026-11-13 03:23:27+00'
WHERE venue='okx' AND surface='dcd' AND template IN ('dcd_put','dcd_call')
  AND enabled=false AND disabled_reason IS NULL;

UPDATE fin_opp_template SET
  disabled_reason='能力债回禁用——启用模板14d零评估产出（工单#13096/升件#13156-13157）；同面 dcd 族 venue 缺类定案同因（canon:dcd-exit-venue-capability-immature）',
  ban_premise='okx DCD 产品面（/api/v5/finance/sfp/dcd）venue 能力缺类/不可用——产品目录与订单面不供给',
  ban_premise_kind='venue_capability',
  banned_at='2026-11-13 03:02:35+00'
WHERE venue='okx' AND surface='dcd' AND template IN ('vol_sell_strangle','vol_sell_single_call')
  AND enabled=false AND disabled_reason IS NULL;

UPDATE fin_opp_template SET
  disabled_reason='测试面登记——dcd-test 扩张自证锚不真探（md_1044⑥；非钦定禁用）',
  ban_premise_kind='none'
WHERE venue='okx' AND surface='dcd-test' AND template='dcd_call'
  AND enabled=false AND disabled_reason IS NULL;
UPDATE fin_opp_template SET state='candidate'
WHERE venue='okx' AND surface='dcd-test' AND template='dcd_call'
  AND enabled=false AND state='enabled';

UPDATE fin_opp_template SET
  disabled_reason='未收养候选——非钦定禁用（收养判据待评估）',
  ban_premise_kind='none'
WHERE enabled=false AND state='candidate' AND disabled_reason IS NULL;

UPDATE fin_opp_template SET state='disabled'
WHERE enabled=false AND state='enabled' AND ban_premise_kind='venue_capability';

ALTER TABLE fin_opp_template VALIDATE CONSTRAINT fin_opp_template_state_check;
ALTER TABLE fin_opp_template VALIDATE CONSTRAINT fin_opp_template_ban_reason_check;

CREATE OR REPLACE VIEW fin_opp_template_roster AS
SELECT id, template, venue, surface, enabled,
  CASE
    WHEN enabled THEN 'live'
    WHEN state='candidate' THEN 'candidate'
    WHEN state='retired' THEN 'retired'
    ELSE 'disabled'
  END AS std_state,
  disabled_reason, ban_premise, ban_premise_kind, banned_at,
  ban_last_verdict, ban_last_check_at,
  CASE WHEN banned_at IS NOT NULL
    THEN round(EXTRACT(EPOCH FROM (now()-banned_at))/86400.0, 1) END AS ban_age_days
FROM fin_opp_template;
