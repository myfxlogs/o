# T-0330 armed_ttl_expired 淤积根修——执行径未建族提案闸前置拒

| 字段 | 值 |
|---|---|
| 状态 | 待验 |
| 立项 | `[角色:COS]` devin @ 2026-11-04（业主派单原话落票：dcd 归因实证） |
| 领取 | `[角色:施工]` devin headless @ 2026-11-04 |
| 优先级 | P2——评估漏斗卫生件：364 条 armed 提案等死淤积 |

## 病灶（业主 dcd 归因实证在案）

评估侧对**执行径未建族**照产 armed 提案——`xvenue_carry` 族
armed→open 显式禁（T-0144/T-0145 未派案在档），提案进 armed 态
后无径可去，淤积至 ttl 过期。

**归因分布（dcd 实录）**：`eval_rejected` 850 / `armed_ttl_expired`
364 / `legacy` 909——armed 等死占非零大头，漏斗计数被幽灵淤积
污染。

## 修法（业主钦定二向，施工裁一）

- **甲**：提案闸前置拒——提案生成时查执行径族白名单，未建族
  direct `eval_rejected`（带拒因族码），不进 armed
- **乙**：模板级 `untradeable` 名单——模板卡挂「无执行径」标记，
  评估侧对标记模板零产

## 工作面（钦定范围）

- 查 `fin_opp_eval`→`fin_proposal` 模板白名单面——执行径族册在哪
  登记、armed→open 门禁在哪判
- 淤积归因分类面实证随票（dcd/查询实录）
- **禁动执行径**——本票只修评估侧前置拒/名单，开径与否归业主裁

## 验收锚

- [x] `xvenue_carry` 新提案不再进 armed（桩测断言：`TestExecFamilyMissingFrontReject`——mt5 能力面在册全谱过阈，eval 仍落 `rejected/exec_family_missing`、fin_proposal 零行；xvenue-carry 本就不入 `paperTemplates` 门闸（armed 本不可达），前置拒在更上游断源——连 fin_proposed 都不进）
- [x] 拒因可溯源——前置拒件带族码归因：`fin_opp_eval.verdict='rejected'` + `evidence->>'reason'='exec_family_missing'`，族码=本行 `template` 列（`SELECT template,count(*) FROM fin_opp_eval WHERE verdict='rejected' AND evidence->>'reason'='exec_family_missing' GROUP BY 1` 即得族分布）
- [x] 既有 364 armed 淤积件处置口径落票（**自然过期放**——施工裁：生产实查 364 件已全部 `fin_retired`（armed_ttl_h=4h 卫生闸已自行排干，当前 armed/proposed 活态零行），死因为 armed_ttl_expired 如实留史不批量改写——改写=清账假绿违例；且归因构成为 funding-hedge 319/perp_quarter_spread 22/quarter_basis_long 22/surface-gap 1，前三族=有径族正常 armed 消费后账龄退役，非"无径可去"淤积——xvenue_carry 生产零提案行（仅 3 rejected eval），票面"进 armed 淤积"前提与生产实况不符已核实）
- [x] 修向记录在案（**甲·提案闸前置拒**选定——见下「修向记录」）

## 修向记录（施工裁）

**选定=甲（提案闸前置拒），弃乙（模板级 untradeable 名单）。**

取舍：甲直接命中病灶——提案根本不进队列（eval 即拒），
乙（模板卡挂标记+评估侧零产）在评估层即绝提案，但需新增模板级
标记面（fin_opp_template/卡标记列）跨多臂读取=面更宽、新增登记
面第二真源。甲只需一处共享谓词，落地为 `internal/platform/execfam`
唯一族册——`lifecycle.admitProposals` 模板门闸（原 `paperTemplates`
册归并同读，门禁逻辑零改）与六面提案写径（hypothesis xcarry/
stakecarry、dcd persistEval、gapscan generateCandidate、funding
persist、grammar basis/volsell）同册前置拒，扩族=execfam 一处登记
双侧零漂移。armed→open 门禁（opener.go xvenue-carry 显式禁）未触。

执行径族册登记处（工作面答）：原 `paperTemplates`（lifecycle/run.go）
→ 今 `internal/platform/execfam`（六族同集）；armed→open 门禁判处
=`openLive`（opener.go xvenue-carry 显式禁）+ `hedgeOpener.Open`
switch 派发（cmd/core/open.go）。

淤积归因分类面实证（SG 隧道 127.0.0.1:5544，2026-11-04 查）：
`armed_ttl_expired` 364=funding-hedge 319+perp_quarter_spread 22+
quarter_basis_long 22+surface-gap 1（与票面 364 数吻合）；
`eval_rejected` 850、legacy_unrecorded 909 同吻合；
xvenue-carry eval 仅 3 行全 rejected、提案 0 行——票面"armed 淤积"
主因实为有径族 armed→消费→4h ttl 退役账龄行（armed_ttl_h=4），
非 xvenue_carry；前置拒落位后未建族提案零产，漏斗计数不再被
幽灵 proposed/armed 位污染。

## 范围外

- `xvenue_carry` 执行径建设——未派案维持（业主红线面）
- armed→open 门禁本身的逻辑改——只加前置拒不动既有门禁
