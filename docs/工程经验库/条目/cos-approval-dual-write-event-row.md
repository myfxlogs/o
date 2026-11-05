---
schema_version: 2
id: cos-approval-dual-write-event-row
kind: pitfall
risk: medium
status: verified
keywords: [ops_event, 已批行, 队列推导, 审批双写, 静默积压]
scope: [process, cos, dispatch, ops_event]
summary: 审批=双写（票面状态+ops_event已批行）——只改票面队列推导面不可见，票静默积压DM无票可派
first_seen: 2026-11-05
last_seen: 2026-11-05
evidence_refs:
  - type: ticket
    ref: T-0370/T-0371/T-0373/T-0374
---

## COS 审批漏写已批事件行=队列静默积压

- **实证**：COS 例行代理批 T-0370/0371/0373/0374 四票——commit
  改票面状态即以为批完；dm_queue.sh 队列推导源=ops_event
  最新刻事件（非票面状态字段），`已批` 行缺席=票在队列面
  不可见，DM 连续数拍无票可派。补写四行后 dm_queue 立见
  全四票。
- **教训**：审批仪式=**双写**——票面状态字段 + ops_event
  `已批` 行同拍落（写面判据 `草案→已批` 归 COS 含事件
  写面）。票面是审计面、事件是推导面——只写一面=另
  一面盲。同型病灶与 T-0368 usage 写径同族：写径不止
  一条时，「做了」的判据是全写径都落。
- **修法候选**（未落，立案参考）：审批小件脚本封装双写
  （approve_ticket.sh——sed 票面+INSERT 已批行+commit 一
  令三件）；或 dm_queue 推导面兼容票面态扫描兜底（两源
  对拍，异源=升件）。
