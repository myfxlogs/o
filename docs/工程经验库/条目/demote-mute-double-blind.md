---
schema_version: 2
id: demote-mute-double-blind
kind: pitfall
risk: high
status: verified
keywords: [降级册, selfEscDemote, 探针降级, 双盲, learn_gap, 升件静默, 分级复升, O-32295]
scope: [cmd/core/selfesc_meter.go, cmd/core/selfgap_esc.go, escEdgeUV 漏斗]
summary: 自观探针降级册「只计不升」全禁言无复升阈——真滞信号与噪声同被吞，滞票44h+无人知；降级必须分级（噪声窗日计+真滞阈升件），禁全有或全无
first_seen: 2026-11-23
last_verified: 2026-11-23
skills: []
evidence_refs:
  - type: code
    ref: cmd/core/selfgap_esc.go learnGapEscHotH/learnGapEscTicketN 分级阈+learnGapCountEdge 计数带死沿（工单O-32295）
  - type: ops_event
    ref: "#26142 求知票滞44h+、#32020/#32021 滞4h+——降级期只日计无人知（spec bug 实证）"
  - type: ops_event
    ref: "#22578① 降级裁——7d 升件约2/3 自观类噪声掩真钱事故3条（降级立论依据）"
---

# 降级册全禁言=双盲——降级须分级复升（O-32295）

## 病灶

#22578① 把自观类探针整册降级「死回合沿只日计一火」——立论正确
（7d 升件 ~2/3 自观噪声掩真钱事故），但实现是**全有或全无**：
册内信号永不升件。learn_gap 实证双盲叠加——求知票滞 44h+（#26142）
与滞 4h+（#32020/32021）只在日计里记数，叠加 dispatch 方言坑致
拾取面瞎，滞票无人知。降级消噪的同时把「真滞」也消音了。

## 判据

凡「降级/静音」类机制，必须自带**复升阈**——噪声窗内静默 +
越界真信号照常升件。全禁言的降级=把探针从「吵」变「哑」，
哑探针比吵探针更危险（假净信号面）。

## 修法（O-32295 落地形）

- 移出降级册（`selfEscDemote` 删键）→ `escEdgeUV` 恢复真升件径；
- 探针侧自分级：计数带（噪声窗内）直 `selfEscMeter.bump` 同键
  日计——降级语义保留；热带（真滞阈）走升件径；
- **计数带死沿与升件态死沿分键**（`learnGapCountEdge` atomic
  vs `deadSince`）——计数带不产恢复行，否则复绿会对从未升过件
  的信号写「恢复自愈」假账；
- 阈值写死常量注释可调（仓内判据钉值惯例——PULSE_TICKS/
  STALL_MIN/TRAIN_MAX 同型），不入 config。

## 教训

- 降级册改动=双向风险：移出=复升，留册=禁言——复升必须带阈，
  否则等于没修；
- 「只计不升」的计数面与「升了再复绿」的恢复面共用 deadSince
  会互相污染——分键是干净解；
- 处置面文档（runbook/playbook）与降级册必须同步改注——册内
  名单是处置席的判据输入，漂移=判词错位。
