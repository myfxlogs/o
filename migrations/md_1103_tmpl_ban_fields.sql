-- md_1103: fin_opp_template 禁因机读化三件套+名册视图（工单O-33393——
-- 业主钦定「策略/模板列表+标准状态，退掉须注明退因；真退体还是误杀」：
-- enabled=f 的理由只活判词行/canon 不查账不可见；enabled∧state='enabled'
-- 双字段矛盾态一次清；candidate 未收养与钦定禁用分态）。
--   disabled_reason=禁因人读（失考也显式标「禁因失考」非留 NULL）；
--   ban_premise   =可证伪命题文本（复核拍重核对象）；
--   ban_premise_kind=命题类别机读键（venue_capability/apr_regime/
--     owner_order/none——复核拍按类派核径，owner_order=业主线不机核）；
--   banned_at     =禁日（龄报锚）。
-- 正本=PG-md（T-0167 分链——fin 链 fin_0234 同形镜像，projector 列携）。
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

-- state 闭集扩 'disabled'（钦定禁用态——enabled=现役/candidate=未收养/
-- disabled=判词禁/retired=族终裁；enabled∧state='enabled' 矛盾态本迁移清零）。
ALTER TABLE fin_opp_template DROP CONSTRAINT IF EXISTS fin_opp_template_state_check;
ALTER TABLE fin_opp_template ADD CONSTRAINT fin_opp_template_state_check
  CHECK (state IN ('enabled','candidate','disabled','retired')) NOT VALID;

-- 钦定禁用行必填禁因（「退掉须注明退因」机械闸——state='disabled' 无
-- disabled_reason=拒写；candidate/retired/enabled 态不受此约，候选行
-- 由 grammar 写径自带「未收养候选」因）。NOT VALID 挂→回填毕 VALIDATE
-- 两段式（工单#25799⑩ 存量校验分离扫）。
ALTER TABLE fin_opp_template DROP CONSTRAINT IF EXISTS fin_opp_template_ban_reason_check;
ALTER TABLE fin_opp_template ADD CONSTRAINT fin_opp_template_ban_reason_check
  CHECK (state <> 'disabled' OR disabled_reason IS NOT NULL) NOT VALID;

-- 回填①钉定禁用四行（dcd_put/dcd_call@dcd + vol_sell_strangle/
-- vol_sell_single_call@dcd——判词锚 ops_event#13096/#13156-13168/#13201，
-- 底判=MANAGE-151 venue 物理缺类定案 canon:dcd-exit-venue-capability-immature）。
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

-- 回填②dcd-test 测试面登记（md_1044⑥扩张自证锚——非钦定禁用，state 归 candidate）。
UPDATE fin_opp_template SET
  disabled_reason='测试面登记——dcd-test 扩张自证锚不真探（md_1044⑥；非钦定禁用）',
  ban_premise_kind='none'
WHERE venue='okx' AND surface='dcd-test' AND template='dcd_call'
  AND enabled=false AND disabled_reason IS NULL;
UPDATE fin_opp_template SET state='candidate'
WHERE venue='okx' AND surface='dcd-test' AND template='dcd_call'
  AND enabled=false AND state='enabled';

-- 回填③candidate 未收养行（非钦定禁用——「未收养」与「禁用」分态显式）。
UPDATE fin_opp_template SET
  disabled_reason='未收养候选——非钦定禁用（收养判据待评估）',
  ban_premise_kind='none'
WHERE enabled=false AND state='candidate' AND disabled_reason IS NULL;

-- 矛盾态清零：钦定禁用行 state 归 'disabled'（enabled=f∧state='enabled' 归零）。
UPDATE fin_opp_template SET state='disabled'
WHERE enabled=false AND state='enabled' AND ban_premise_kind='venue_capability';

-- 回填毕存量校验（两段式下半——NOT VALID 约束转正）。
ALTER TABLE fin_opp_template VALIDATE CONSTRAINT fin_opp_template_state_check;
ALTER TABLE fin_opp_template VALIDATE CONSTRAINT fin_opp_template_ban_reason_check;

-- 名册视件（业主一眼面——「禁了什么/为什么禁/禁了多久/前提成不成立」
-- 一条 SQL 答得出；evalqueue_face 名册段同读此面）。
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
