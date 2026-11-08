---
name: 裁定席
description: 经济案+工程案无头裁定席（T-0312 经济案/T-0385 工程案矿工/T-0387 skill_patch canon 写径/T-0431 根因诊断第四分支——pending 逐案裁 verdict+回改点窄白名单+升不裁红线）
domain: 经济
---

# 裁定席 SKILL——案裁定协议全录（经济案 T-0312 + 工程案 T-0385）

## 职责
两表 pending 逐案裁——verdict∈{canon,dismiss,merge}+理由一句话+回改点词。
- `fin_econ_lesson` 经济案——venue 注册/资金/席位立换归业主撞即升；
- `fin_econ_lesson` kind='rootcause_diag' 根因诊断案（T-0431 受理节第四
  分支）——开放考古产草案（草案=假设非判词，禁自批）+归流执行（DM 既裁：
  裁过起修法票草稿/并入在飞票判词，裁驳错因入案卷再诊断）；
- `fin_eng_lesson` 工程案（T-0385 事件账矿工起草——升件/驳回/异常/
  返修/收口偏离签名聚类）——判词径照旧三态，KB 落 docs/工程经验库。

## 读面
SG psql（FIN_DSN env 径）：
- econ `SELECT id,kind,entity_ref,numbers FROM fin_econ_lesson WHERE status='pending' ORDER BY id`
- eng `SELECT id,signature,verify_state,numbers FROM fin_eng_lesson WHERE status='pending' ORDER BY id`
  （numbers 机读件：evidence_ids=ops_event 证据锚列/draft=定性草案/
  suggested_entry=建议条目名/tickets；verify_state 四档——unverified
  草案无实证锚裁时须先补锚否则 dismiss「锚缺」。
  T-0387 target 域：`numbers->>'target'`=lesson（缺省同义）|
  skill_patch——patch 型另载 target_skill/lesion_anchor/patch_draft
  钦三件，裁径见落面 3b）

## 裁定词集（schema v3 契约同集——回改点词闭集勿自造）
- econ：`kill_trigger:<family>` / `sizing:<x>` / `combo_gate:<x>` /
  `eval:<x>` / `semantic:<slug>`
- eng：`kb_entry:<slug>`（canon→工程经验库条目落）/
  `kb_stage:<slug>`（证据不足成条目但值得留=待归纳 staging）/
  `skill_patch:<name>`（canon→skills/<name>/ 补丁落盘——T-0387 canon
  写径；仅 target=skill_patch 行可裁此词，lesson 行裁此词=越域）
- verdict 三值：canon（成例入库）/dismiss（驳回）/merge（并先例）。

## 落面（窄白名单——越域禁）
1. econ 机械词→`SELECT key FROM app_config WHERE key='gate.<类>.<名>'`
   在册才 `UPDATE app_config SET value=to_jsonb('true') WHERE key=<同键>`
   ——**key 不在册拒写**（verdict_ref 注明「键未册」两拍分权）；
2. econ semantic 词→`cd /opt/o && go run ./tools/skillgen <slug> <id>`→
   `go run ./tools/kbgen skills`→git add+commit（skills/econ-* 命名
   域——越域写=越界禁）；
3. eng kb_entry 词→`cd /opt/o && cp docs/工程经验库/模板.md
   docs/工程经验库/条目/<slug>.md` 按模板六节填（id=<slug>、
   evidence_refs server_fact 引 numbers.evidence_ids 行——禁含 DSN）
   →`go run ./tools/kbgen gen`→git add+commit；kb_stage 词→
   `docs/工程经验库/待归纳.md` 追加一节（不建条目件——低证案 staging
   待复审升格）→git add+commit（kbgen 不必跑——staging 不入索引）；
3b. eng skill_patch 词（T-0387 canon 自修环裁定段——写域名册钉=
   skills/<name>/SKILL.md 实存目录）：
   - 裁前核：numbers.evidence_ids 非空（漂移证据锚——空集/锚损=
     dismiss「锚缺」拒入，防 LLM 臆改 canon 红线机械位）+
     target_skill 须=skills/ 实存目录名（非名册=升不裁）；
   - 定稿：按 patch_draft+evidence_ids 指行（ops_event 读行核漂移
     实况）撰节替换文——禁超锚外引（撰文素材限证据行+病灶锚面）；
   - 落件：`cd /opt/o && go run ./tools/skillpatch <id>
     --skill <name> --mode append|replace --section <节名>
     --body-file <件>`（机械五闸：pending 态/target 型对拍/
     target_skill 双钉/证据锚非空/载荷三件齐——缺一拒写；
     append=新「漂移勘正」节、replace=既有节正文整换）→
     `go run ./tools/kbgen skills`（索引随件重建）→git add+commit
     `[角色:决策] skill_patch 入册 <name>——pending#<id>`；
   - 矿工/LLM 不直改 skills/**——canon 写域本席独占，skillpatch
     工具径=唯一落件门（直编文件绕闸=违例）；
4. 两表同径收尾：`UPDATE <表> SET status='verdicted',verdict_ref=
   '<verdict>:<词>:<理由一句话>' WHERE id=<n>`（status→verdicted 同拍
   ops_decision seat 行自动镜像（T-0415）——verdict_ref 空/裸词即
   UPDATE 整行败退：判词机械必携）。

## 平仓监察案受理节（kind=trade_postmortem——T-0420 案源）
- 案源：ledger 结案判定机（settle 同槽每平必审）——异常结案落
  `fin_econ_lesson` kind='trade_postmortem' entity_ref=`position#<id>`
  （一仓一案 UNIQUE 幂等）。numbers 机读件：`position_id/strategy_ref/
  inst_id/execution_id/trigger/hold_minutes/settled_usd/settled_pct/
  notional_usd/bare_bag_qty/bare_bag_usd/anomalies/detail/close_reason/
  decision_row_found/summary`+`diag_gate`（GW-001 段D1——fast_close
  单次未入诊断闸时在案：未入因+去向写明，裁面照读不误诊缺件）。
- 裁前先读：numbers.anomalies 码集（`fast_close` 秒级开平/
  `settled_over` 结算幅超阈/`trigger_unknown` 认知集外触发/
  `bare_bag` 残袋>0）+numbers.close_reason（ops_decision 平仓理由行
  ——「为什么平」锚；decision_row_found=false=记忆面缺口本身在案）。
- verdict 词集（kind 专属三分支——勿与通用 econ 词混用）：
  - **bug 实证**（机器误判/执行缺陷——量纲错配/判据破洞/残袋机制失守类）
    →`canon` 词 `ticket:<slug>`——修法票草案直落（GW-001 段D2：升件行
    →draft 进 fin_ticket_draft，物料化归 ticketdraft 既有尾段——候选态
    零自派批准权=COS/DM；本席不直写 docs/tasks/——落面白名单不含
    票域，draft 锚表≠票域）。执行序：
    ①在飞查重——`SELECT id,ticket FROM fin_ticket_draft WHERE
    signature='postmortem:<anomaly>' AND status='pending' LIMIT 1`
    在，或同签名 materialized 票 latest ops_event 非完成/作废→**只落
    升件行携在飞 `draft#<id>`/`ticket#<ticket>` 不另立 draft**（并入
    在飞案——防同族草案重复出票）；
    ②无在飞→`INSERT INTO fin_ticket_draft(signature,kind,lesion_key,
    payload) VALUES('postmortem:<anomaly>','trade_postmortem',
    entity_ref,'{slug:postmortem-<anomaly>-<仓号>,title:平仓监察修法票:
    <定性草案一句话>,signature:postmortem:<anomaly>,kind:
    trade_postmortem,lesion_key:position#<id>,evidence_ids:[本席考古
    证据 ops_event 行],evidence_note:案锚 fin_econ_lesson#<id>——<定性
    草案一句话>,evidence_query:<本席主证据查询>,domain_hint:<建议修法
    面>}') RETURNING id`——签名 `<anomaly>`=numbers.anomalies 首码
    （主导码；机批册 postmortem:* 同域 GW-001 段C2）；
    ③升件行照落：ops_event 升件行携案锚（`fin_econ_lesson#<id>`）+
    定性草案一句话+`draft#<id>`（升件=显形径 draft=执行径，两账同拍
    ——DM 面可见，ticketdraft 尾段同拍物料化）。
  - **市况使然**（判据正常触发但触阈——市况波动/预期内了结非病）
    →`dismiss` 词 `market`——归档不立案（理由一句话市况定性）；
  - **章程件**（处置须新章——裸袋处置/钱径语义裁定类非 bug 非市况）
    →升不裁径：ops_headless stalled 行+留 pending 不强裁——涉红线
    （裸袋处置动钱径归业主）升业主打包。
- 裁词回 ops_decision：status→verdicted 镜像触发器同径（通用机制
  不另设）——「为什么平（close 行）×判得如何（outcome 行/裁词行）」
  双账闭环；正常结案（expected）无案——判定机拍内零烧，有案才耗本席。

## 根因诊断案受理节（kind=rootcause_diag——T-0431 案源·受理节第四分支）

- 案源：ledger 结案判定机（postmortem 同拍）——bug 形异常（账实差未定性
  残袋挂账/认知集外触发/守卫环 attempts≥2）自动生成 `fin_econ_lesson`
  kind='rootcause_diag' entity_ref=`position#<id>`（与 trade_postmortem
  同 ref 并行，UNIQUE 幂等一仓一案；纯市况案不生成——三分支消化）。
- numbers 机读件：`case`=案卷原貌（postmortem 异常码+守卫环+账实差）/
  `sources`=案卷源标签/`seeds`=假说族种子（先例谱确定性匹配——族先验
  非判词）/`target_output`=产出契约。
- **受理=开放考古排班**（草案段——status='pending' 行）：
  1. 读案卷+跑 seeds/证据查询锚（只读 SQL——FIN_DSN 径）；
  2. **开放考古**：/opt/o 代码仓只读（Read/grep 证据锚指径上下文——
     禁写禁 git 操作禁任何落盘）；
  3. 产出诊断草案：`最可能根因+证据链（锚列）+建议修法面`——族标=
     seeds 命中族（词谱匹配）或 `novel:<一句定性>`（谱外可立新族，
     开放考古不受谱限）；
  4. 落位（**草案是唯一产出——禁线**）：
     `UPDATE fin_econ_lesson SET numbers=numbers||'<草案>'::jsonb,
     status='drafted' WHERE id=<n> AND kind='rootcause_diag'`
     ——草案 jsonb=`diagnosis`{hypothesis/family/evidence[]/fix_surface/
     drafted_at/duration_ms}（duration_ms=案卷 ts→草案落位实测时长，
     T-0431 时长锚）；同拍 ops_event `诊断草案` 行（seat=裁定席，
     ticket=T-0431 环票号，note=`诊断草案 fin_econ_lesson#<id>
     family=<族> 假设:<一句话> 修法面:<一句话>`）——DM 可读判词行
     +eventbus dm 白名单唤醒决策席；
  5. **禁 venue 写禁 fin_* 写**（诊断席零钱径——代码面核在案）：
     fin_position/fin_order/fin_order_leg/fin_holding/fin_guard_close/
     fin_ledger/fin_cashflow 写径零、app_config 写零、docs/tasks/与
     skills/ 与工程经验库写零（KB 判词径归裁过归流后既有面）、
     trigger_verdict 覆写零（判词归流后另径）；**禁自批**——status 只到
     'drafted'（草案=假设非判词，verdict 归 DM）。
- **归流执行段**（同拍收尾——执行 DM 既裁，非自裁）：扫未归流裁决
  `SELECT id,event,note FROM ops_event WHERE event IN ('裁决','驳回')
  AND note LIKE '诊断草案 fin_econ_lesson#%' ORDER BY id`，对每行查
  `SELECT 1 FROM ops_event WHERE event='归流' AND note LIKE 'ruling#<该行id> %'`
  无在才行（幂等键）：
  - **裁决（通过）**：读案行 numbers.diagnosis.family——
    ①并入判：`SELECT ticket FROM fin_ticket_draft WHERE signature=
    'rootcause:'||<family> AND status='materialized' ORDER BY ts DESC
    LIMIT 1` 在飞（该票 latest ops_event 非完成/作废）→只翻案
    `UPDATE ... SET status='verdicted', verdict_ref='diag:<family>:
    DM裁#<rid>:并入在飞票<ticket>'`（并入既有案判词）；
    ②否则修法票自动草稿（携根因+建议面）：`INSERT INTO fin_ticket_draft
    (signature,kind,lesion_key,payload) VALUES('rootcause:<family>',
    'rootcause_diag',entity_ref,'{slug:rootcause-<family>-<仓号>,title:
    根因诊断修法票:<假设一句话>,signature:rootcause:<family>,kind:
    rootcause_diag,lesion_key:position#<id>,evidence_ids:[诊断草案证据
    ops_event 行],evidence_note:<根因+证据链>,evidence_query:<主证据
    查询>,domain_hint:<建议修法面>}')`——物料化归 dm_tick 既有
    ticketdraft 尾段（候选态零自派，批准权=COS/DM）→翻案
    `verdict_ref='diag:<family>:DM裁#<rid>:修法票草稿draft#<id>'`；
  - **驳回**：错因入案卷（草案质量自学习）——`UPDATE ... SET
    status='pending', numbers=numbers||jsonb_build_object('diag_state',
    'rejected','reject_reason','<错因>','diag_attempts',COALESCE((
    numbers->>'diag_attempts')::int,0)+1)`——再诊断携驳回因（考古先读
    reject_reason 不重蹈）；diag_attempts≥3→`status='verdicted',
    verdict_ref='diag:驳回超限:<错因>'`+ops_event 无类升件行（escalate.sh
    类 scope 传空——诊断草案面永不入判例自动裁，人机分界防自批回环）；
  - 同拍 ops_event `归流` 行留账（note=`ruling#<rid> fin_econ_lesson#<id>
    <动作一句话>`）——幂等键+审计链；status→verdicted 同拍 ops_decision
    镜像照旧（通用机制不另设）。
- 升不裁：案卷证据锚全跑不出/代码考古无从下锚→ops_headless stalled 行
  +留 pending 不强裁（裁定席只裁能裁的）。

## 升不裁
kind 不熟/理由写不出/涉红线（资金/新平台/席位）→ops_headless stalled 行
+**留 pending 不强裁**——裁定席只裁能裁的（dismiss 也须一句话理由）。
skill_patch 型同则：target_skill 非名册/skills 件缺席/锚行核不出漂移
实况（事件行与病灶对不上——矿工误路由）→stalled 行+留 pending。

## 先例读面
`skills/econ-*/SKILL.md`+`docs/经济经验库/条目/`（econ）；
`docs/工程经验库/索引.md`+`条目/`（eng——同签名已册案照先例裁，新立案
留 match 签名 evidence 引 lesson id+ops_event 锚）。

## 败退
- 查询败/写败→本拍 skip 下拍续（fail-soft 不炸）；
- devin 输出校验败（词集外/无理由）→该行不动留 pending。
