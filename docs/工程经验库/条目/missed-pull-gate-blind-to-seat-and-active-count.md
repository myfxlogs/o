---
schema_version: 2
id: missed-pull-gate-blind-to-seat-and-active-count
kind: pitfall
risk: high
status: verified
keywords: [漏拉闸, dm_tick, ops_event, 派单, seat, 并行上限, o-worker, systemd-run, T-0329]
scope: [dm_tick.sh, 漏拉清扫闸, ops_event 事件面, 拉起式施工]
summary: 事件闸只认事件词不认行为者、不设容量闸=双盲——任何席位写一行 `派单` 词（含批注误用/通报用义不同者）即被闸当派工令拉起工人，且不数活数可突破并行上限。事件驱动的拉起/执行类闸必须同时钉「源白名单（seat）+容量上界」，词义消歧另走词表规。
first_seen: 2026-11-04
last_verified: 2026-11-04
skills: []
evidence_refs:
  - type: server_fact
    ref: 2026-11-04 ops_event id 1562/1563 均 seat='COS' event='派单'（「已批入队」批注误用派单词）——被 dm_tick 漏拉闸当决策席派工令拉起 T-0326/T-0327 双工，三工并行破当时上限=1（活无撞，机制越界实锤）
  - type: doc
    ref: docs/tasks/T-0329-漏拉闸活数盲-补拉不计并行上限与派单源分源.md
  - type: doc
    ref: scripts/meta/dm_tick.sh（修后：EXISTS 加 seat='决策席' 谓词 + list-units 活数≥2 break 空转）
---

# 漏拉闸双盲——不认派单源 seat、不数活数（T-0329）

## 坑形

事件驱动闸以「最新事件=某词」为触据：`ops_event` 里谁都能写
`派单` 行（report.sh 事件词集跨席共用），闸不看 `seat` 列；且
闸按「票」逐条拉起、全程不数已活 `o-worker-*` unit 数。

后果二重：
1. **源盲**——COS 把「已批入队」批注错用 `派单` 词（词义同名
   异物：通报 vs 派工令），闸照拉；
2. **量盲**——每张滞留票各拉一工，并行上限形同虚设（上限=1
   时实跑三工）。

零错零告警——误拉起的 run 正常领票施工，账面完全合法。

## 根因

事件词≠身份。`event` 是**词表**，`seat` 才是**行为者**——凡
「某事件触某动作」的闸，谓词必须同时绑事件词+许可行为者集；
凡消耗共享容量（进程/连接/钱）的自动拉起，必须配容量上界闸
（先数后拉）。

## 修法（T-0329 已落）

- 扫票 SQL `EXISTS` 谓词加 `AND e.seat='决策席'`——仅决策席
  常径机派行触闸；
- 拉起前 `list-units 'o-worker-*' --state=active,activating`
  计数≥并行上限即 `break` 本拍空转；
- 语义件同修：COS 职卡钉「批注用 `已批` 词、`派单` 限批复
  转办」——词义分流防再误用（闸已过滤仍防流水语义污染）。

## 判据

凡新写「事件→动作」闸：审——①触据是否绑行为者白名单而非
裸事件词；②动作是否先数容量再消耗。两问任一否=同坑预埋。
