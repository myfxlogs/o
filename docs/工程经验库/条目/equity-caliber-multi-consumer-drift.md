---
schema_version: 2
id: equity-caliber-multi-consumer-drift
kind: pitfall
risk: high
status: verified
keywords: ["dd 硬线", "权益口径漂移", "AttachEquity", "WithEquity", "假回撤", "peak<=0", "dd_no_base", "户间划转", "T-0050", "T-0264"]
scope: [cmd/core, internal/guard, internal/risk, venue 权益现读, okx]
summary: 同一领域量（venue 权益）两个消费面平行接线各定口径——T-0050 把 riskGate 权益扩成交易户+funding+Earn 三分量，dd 硬线 AttachEquity 仍只读交易户：停泊态 trading≈0→peak<=0 硬线空转、户间划转呈假回撤/假增长（误触全平+落闸）。同一领域量多消费面=单源闭包接线，禁平行口径
first_seen: 2026-10-30
last_verified: 2026-10-30
skills: []
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0025-真钱推进执行链.md（审计注记①）
  - type: doc
    ref: cmd/core/venue_readers.go（attachVenueReaders——okxEq 单源闭包两挂点）
  - type: doc
    ref: docs/tasks/T-0264-dd硬线权益基准对齐.md
---

## 事实/症状

`attachVenueReaders` 双权益口径分叉（T-0025 审计注记①实证）：`riskGate.
WithEquity` okx 支=tradingVals+fundSaveBalances 三分量（T-0050 停泊态口径），
`wd.AttachEquity` 挂的 `liveEq` 只读 tradingVals 交易户。两向后果：闲资停泊
Earn 常态 trading eq≈0→dd 基准 peak<=0→dd_no_base 硬线空转（真回撤看不见）；
开仓后户间划转（trading↔funding）在 dd 面呈假回撤/假增长（-3%/30d 误触
全平+kill_switch 险）。

## 根因或策略解释

扩口径票（T-0050）只改了它在改的那个消费点——riskGate；第二消费点 dd 硬线
的平行闭包无人同步。「同一领域量 N 个消费面」结构下，每加一个消费点=口径
漂移多一处发端，且失败形态全是隐性（空转不报错、假回撤像真信号）。

## 处置

T-0264：抽 `okxVenueEq` 三分量聚合包级件（trading+fundSave 读件注入可桩），
`okxEq` 单闭包同挂 riskGate.WithEquity 与 wd.AttachEquity——单口径源、
读败 fail-closed 语义同承 T-0058。判别式：**同一领域量出现第二消费点时，
必须先抽共用件再挂两挂点；发现平行闭包各聚各的=口径漂移病灶**。

## 证据

- check: cmd/core/venue_equity_test.go 三桩测（停泊 Earn>0+trading≈0→
  基准非零；户间划转净值恒稳；任一分量读败=err 上抛）
- result: 3 passed（T-0264 施工支）
- ref: T-0025 审计注记① / docs/tasks/T-0264-dd硬线权益基准对齐.md

## 反证条件

- check: venue 侧若将 dd 硬线钦定为「只盯交易户」口径（与 riskGate 刻意
  异源），本条目失效——彼时须票面明钉异源理由而非默认分叉
- falsifies_when: 决策档钦定 dd 权益口径≠riskGate 口径
