---
schema_version: 2
id: dispatch-seat-whitelist-phase-lie
kind: pitfall
risk: high
status: retired
keywords: [派单, seat白名单, dm_queue, ops_ticket_phase, 相位假象, 补拉窗]
scope: [机制面, 事件写径, seat语义]
summary: 相位视图只看 event 词不看 seat，补拉候选滤 seat∈{决策席,CTO}——非钦定 seat 写派单=票呈 dispatched 但永不拉起
first_seen: 2026-11-09
last_seen: 2026-11-09
evidence_refs:
  - type: provenance
    ref: COS 清账批 seat='COS' 写派单 T-0443/0479/0481/0482/0484→相位 dispatched 而 4h 零拉起，断链探针升件在案（ops_event 交行#4944-4947）
  - type: provenance
    ref: seat='决策席' 重发派单后 dm_queue.sh dispatched 即刻出列五票
---

## 「相位=dispatched」≠「可拉」——派单 seat 是路由键不是署名

- **实证**：`ops_ticket_phase` 视图按 event 词映射（派单→dispatched）
  不滤 seat；`dm_queue.sh` 补拉候选 SQL 却要
  `latest_event='派单' AND latest_seat IN ('决策席','CTO')`。COS 席
  清账以自身 seat 写派单→票相位 dispatched 而拉取窗盲区，5 票
  4h 零拉起；机械轨断链探针正确升件但 seat 语义它修不了；
  决策席 seat 重发即入列。
- **根因**：相位投影宽松（全 seat）与消费面收窄（钦定 seat 集）的
  口径断层——写面须按消费面钦定集署名，不能按「我是谁」署名。
- **对治**：写派单/领单类驱动相位的事件前查消费面 seat 集；
  快检径=`dm_queue.sh dispatched` 出列才作数，相位面单看不作数；
  修后重发以 note 记明代位（台账 attribution 不撒谎）。
