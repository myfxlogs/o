---
name: 裁定席
description: 经济案+工程案无头裁定席（T-0312 经济案/T-0385 工程案矿工/T-0387 skill_patch canon 写径——pending 逐案裁 verdict+回改点窄白名单+升不裁红线）
domain: 经济
---

# 裁定席 SKILL——案裁定协议全录（经济案 T-0312 + 工程案 T-0385）

## 职责
两表 pending 逐案裁——verdict∈{canon,dismiss,merge}+理由一句话+回改点词。
- `fin_econ_lesson` 经济案——venue 注册/资金/席位立换归业主撞即升；
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
