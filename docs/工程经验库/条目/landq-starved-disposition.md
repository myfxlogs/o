---
schema_version: 2
id: landq-starved-disposition
kind: protocol
risk: med
status: verified
keywords: [landq_starved, land_queue, 候审支, 复审过, 绿戳, 滞留支, 双支仲裁, 判废清支, aseat]
scope: [scripts/land_queue.sh, ops_event, git worktree, 唤知会话]
summary: landq_starved 升件=候审支滞留信号，处置径四查（远端可见/戳鲜≤6h/复审过行 sha=全尖/净合），判废清支先例三形（RED断言载体/内容被并支票消费/同工单双支取一）——aseat 散文匹配可被复审行自身词面毒化（候审回执→aseat误取复审席→自锁），机读分席径待修
first_seen: 2026-11-21
last_seen: 2026-11-25
evidence_refs:
  - type: ops_event
    ref: 升件#27728 #27771 #27932 #28944 #29929 #30124 #30323（同窗7件簇）
  - type: ticket
    ref: 工单/landq-aseat-regex-poison（P1 未修）
---

## landq_starved 处置径（候审支滞留四查 + 判废先例）

- **信号语义**：land_queue 每拍扫 fix/feat/t/audit/revfix/hotfix/canon 前缀支，
  三件不齐（鲜戳 `.git/check-green-<全sha>` ≤6h ∨ 复审过行 sha=<全尖> ∨ 净合）
  或面外（无前缀/仅本地）→升 `部署/landq_starved`。滞留集比对节流——新滞留支
  即火、同集 6h 复火。
- **四查径**（判前必走全，勿凭升件词面判）：
  ①远端 `git ls-remote` 支在+尖 sha；②`.git/check-green-<全尖>` 戳龄；
  ③`ops_event` 复审过行「复审过 sha=<全尖>」+seat≠作者席；
  ④`git merge-tree --write-tree origin/master <尖>` rc=0=净合。
- **缺复审=本拍补签**：diff 三维（内部一致/oracle对拍/边界语义）过→
  `INSERT ops_event seat='复审席' note='复审过 sha=<全尖> …三维判词'`，
  下拍 land_queue 自合——**勿手工 merge**（单写者契 D-0048③）。
- **回执缺=第四隐性要件**（2026-11-25 双支实证——戳鲜+复审过+净合
  三查全过仍跳）：`review_ok()` rc3=「完工回执缺/作者席不可证」——
  land_queue 硬闸实为四件非三件：须 ops_event 有回执形行
  （event~'完工|候审|施工收口|工单完工|回执' ∨ note 行首≤40字含
  「完工回执|完工推支候审|候审回执|施工收口」∨ event='处置'∧note
  行首 'O-\d+…@hex{9,40}'）引支名∨尖前9位，写行席≠复审席。
  唤知/处置席直造支收口**必须补写** `seat='处置席' event='处置'
  note='完工回执 fix/<支>@<全尖>——…'` ——作者会话猝死簿记断尾=
  支永久滞留（fix/openingequity-cfd-proto+fix/39718 同窗 1h+ 实证）。
  接续席补落合法（同席族簿记链——evidence 在戳+推支证非虚构完工）。
- **判废清支先例三形**（清远端支销滞留）：
  ①audit/* RED 断言载体支——内容已入升件+工单即判废删远端
    （断言是产物非交付物，同 audit/1121-life、audit/1122-forensic 例）；
  ②支内容已被他支/merge 消费——stale 支删（audit/1121-life→tx-atomic 例）；
  ③同工单双支——逐 diff 比修法覆盖面+oracle 强度取一签复审，
    劣支判废删（fix/deploywatch-esc-sqlquote vs -quote 例：
    前者双参 sqlq 全覆+词法级桩校验=真 oracle，后者只覆单参）。
- **aseat 散文毒化坑**（未修，工单/landq-aseat-regex-poison P1）：
  `review_ok()` 取「完工回执|完工推支候审|候审回执」末行 seat 当作者席——
  复审行 note 含「候审回执在案」即被误判作者席=复审席→复审行 seat<>aseat
  拒收=自锁。即时变通=复审行措辞避「回执」词；根修=作者/复审证据
  机读分席（勿靠散文匹配）——修法候选见票。
- **防坑**：`git merge-tree --write-tree` 只接两支参——传三参错用
  merge-base 语法会 rc≠0 误判冲突（git≥2.38 自动算基）。
