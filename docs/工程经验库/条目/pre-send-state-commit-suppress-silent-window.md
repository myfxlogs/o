---
schema_version: 2
id: pre-send-state-commit-suppress-silent-window
kind: pitfall
risk: high
status: verified
keywords: [发前账律, 窗态持久化, 闭段抑制, SendText败, 静默窗, 水位实证, pending close, feed narrative]
scope: [cmd/core, feed_push, fin_push_ledger]
summary: 窗态在 SendText 前落库+闭段即抑成员行——发送失败后重试时叙事块不再产出而成员行仍被抑=该会话窗内容整段静默消失；闭标消费必须以水位实证为界
first_seen: 2026-11-25
last_verified: 2026-11-25
skills: []
evidence_refs:
  - type: ticket
    ref: O-37510
  - type: code
    ref: cmd/core/feed_narrative_blocks.go
---

## 现象

feed 叙事模式 PhaseA 剔集闭径（唤起果/唤起败——feed 剔集闭标须库查
补解）：`narrativeBlocks` 在返回块**之前**已把闭段写进 `st.Closed`
并把窗从 `st.Open` 除名，`saveNarrState` 发前持久。若随后 `SendText`
失败：水位不进→下拍 `pending()` 重取同批行→窗态里该段已是 Closed——
`inClosed` 继续抑成员行逐行渲，而 PhaseA 不再为它产叙事块（已不在
Open）→**该会话窗整段内容静默消失**，无任何报错无降级。

## 根因

「发前账律」（窗态先落库防崩丢）与「闭段抑制」（Closed 段成员行不二泄）
组合后，闭段登记的**生效时点**早于发送成功的**实证时点**——状态机把
「已决」当成「已达」，中间失败窗内两边都不认账：叙事不重发（已闭）、
逐行不补发（被抑）。同族形态：任何「先提交消费标记、后发文」的管线，
标记必须可区分「已决未达」与「已达」——否则失败重试=静默吞内容。

## 处置（O-37510 落地形态）

- 闭段登记拆两态：`st.Pending`（已决待实证）→`st.Closed`（实证转正）。
  实证锚=`BatchHi`（载叙事批行 id 上界）——`wm≥BatchHi`=该批
  SendText 成功（wm 只在送成后推进，批序单调无跳批）；不用闭标 id
  作锚因闭标行 feed 剔集不入水位域，`wm` 可能永远过不了它。
- 发败（wm<BatchHi）=窗保开下拍重发叙事；实证=转正 Closed+窗除 Open。
- 窗暂留 Open 期间成员行续被 `inOpen` 吸收抑制——不二泄不丢。
- 连带界外修正（同族前病灶）：`inOpen` 增 `cidOf` 闭标界——已决闭标
  界外行不归本窗（旧径界外成员行被吞不逐行=另一静默窗形）；
  PhaseB 会话收口加 `r.id<=cidOf[u]` 闸——前序闭标（唤起果/败）已决
  后会话收口行不重闭走逐行（同产线并存实证在案）。

## 验收

- 桩测 `TestNarrativeBlocksPendingClose` 全链：产出→pending 登记
  （Closed 空/Open 留）→发败重发（叙事再产）→实证转正（Closed 落/
  窗除/pending 清）。
- 语义守恒：穿透集/逐行兜底/feed.narrative_mode 开关零触。
