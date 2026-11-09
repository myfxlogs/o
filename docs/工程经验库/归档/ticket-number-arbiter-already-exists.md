---
schema_version: 2
id: ticket-number-arbiter-already-exists
kind: pitfall
risk: medium
status: retired
keywords: [取号, 票号, next_ticket, 撞号, 存量资产, 立机制前先查]
scope: [流程机器, scripts/next_ticket.sh, scripts/check_ticket_numbers.sh, docs/tasks 取号径]
summary: 立新机制前不查存量资产——T-0373 初稿拟立 PG 序列仲裁取号，施工前复核发现 T-0163 双闸+T-0219 next_ticket.sh 早已保证唯一性，真病灶只是席面不可见（件在而席不知）；改判为职卡钉径+领号台账行，序列案录淘汰
first_seen: 2026-11-05
last_seen: 2026-11-05
evidence_refs:
  - type: commit
    ref: T-0373
  - type: doc
    ref: scripts/next_ticket.sh
  - type: doc
    ref: scripts/check_ticket_numbers.sh
  - type: doc
    ref: docs/collab.md
---

## 病灶

决策席/COS 立票均未走 `scripts/next_ticket.sh`——COS 以 `ls` 本地视野
取号（差一轮未撞属运气）。仲裁件 T-0163（pre-commit `--staged` 同号拦
+pre-push `--remote` 号≤远端最大号拦）与 T-0219（fetch 后扫
origin/master 最大号+1）早已在案且实证根治 T-0154/T-0160/T-0217
三撞——**机制白建在席面不可见**。

## 处置

职卡钉径非新机制：决策席 SKILL 形态域界节+COS SKILL 新增取号径节，
明写「取号唯一径=scripts/next_ticket.sh <席位名>，禁本地视野扫描」；
next_ticket.sh 分配成功顺手写 ops_event `领号` 行（seat 入参，
台账审计件非硬依赖——DSN 缺席/写败 `|| true`+# tolerate: 锚静默退
不阻断取号）。PG 序列仲裁案评估淘汰在档（唯一性推送闸已保证，
序列只省偶发重编却引入 DB 可达性依赖——勿再开题）。

## 证据

- check: 桩测 `scripts/meta/tests/` 领号行场景——psql 桩捕获 INSERT
  断言 seat/event/ticket 三列；DSN 缺席径取号输出不受阻断
- result: 全谱实录见 T-0373 票面回填
- ref: T-0373

## 反证条件

- check: `git log` 找「立新取号/票号机制」类票面——若其论证未先引
  next_ticket.sh+check_ticket_numbers.sh 存量件即属同坑复发
- falsifies_when: 双闸/next_ticket.sh 被证实存在系统性漏号（如远端
  max 计算被绕过），此时「件在」前提崩塌须重评序列案——当前无此证据

## 适用边界

凡「要立机制解决 X」前置动作=先 `grep`/`ls` 存量资产面（scripts/、
skills/、docs/collab.md 条文）——本案病灶非代码缺陷是认知缺口：
仲裁件存在但职卡未钉=等于不存在。推及一切「看似缺机制」立票。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a（豁免由：流程认知坑非产线数据面——无适配 gauges.tsv
  表，复发监测=ops_event 领号行台账人工抽阅+撞号闸拒绝流水）
