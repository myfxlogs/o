---
schema_version: 2
id: dual-listenloop-serial-starvation
kind: pitfall
risk: high
status: verified
keywords: [ListenLoop, 阻塞死循环, 串行调用, 监听臂零跑, ev_funding, WaitForNotification]
scope: [internal/platform/events, internal/research/hypothesis, 事件驱动臂]
summary: ListenLoop 阻塞死循环——Run 内任一同步调用都占死其后全部行（双听须皆 go）；首轮返修只给第二听加 go=没修，「修症状行不修根因行」实证；验收须看监听注册+可观测产出，码面看着对不算数
first_seen: 2026-10-26
last_verified: 2026-10-26
skills: [生产取证]
evidence_refs:
  - type: code_fact
    ref: internal/research/hypothesis/run.go:104-109——双 ListenLoop 串行，research-hypo(ev_gap) 占死致 research-hypo-sig(ev_funding) 零跑（缺 go 实证）
  - type: server_fact
    source: postgres
    ref: md_funding +672行/45s 事件流正常 vs fin_opp_eval xvenue-carry 零行（T-0144 验收窗 2026-10-26）
---

# 双 ListenLoop 串行=第二听占死

## 病灶
`events.ListenLoop` 是阻塞死循环（WaitForNotification 常驻）。Run() 内顺序
调两次=第二听永不到行——`research-hypo-sig`(ev_funding→ScanSignals) 自
T-0140(2bb5751) 起零跑，被 `research-hypo`(ev_gap) 首听占死。**验收难捕面**：
臂无产出与「无卡如实」表象全同——T-0140 验收以「零卡=正确」通过，T-0144
评估行该落而未落才暴露。

## 判据（排查序）
- 监听臂零产出先查 ListenLoop 是否 `go` 启动/独占阻塞位——同一 Run 内
  任一 ListenLoop 同步调用都占死其后全部行，**不只第二听**。
- 修饥饿类病灶=给**首个阻塞调用**加 go（或全加），给死代码行加 go=零效果。
- 验证事件链三层分验：trigger 在库 ≠ NOTIFY 发出 ≠ 监听臂跑到 fn；
  监听注册实证=pg_stat_activity `LISTEN <ch>` 连接计数按臂名对账。
- 「无卡=正确」类验收须先证臂在跑（心跳/计数器/日志痕），缺席证据≠阴性。
- **修后验收必须复跑原始阴性证据**（上轮缺的闸）——本轮首轮返修
  commit 后未查 pg_stat_activity/评估行就部署，病灶原封进生产半天。

## 实证
- T-0144 验收窗（2026-10-26）：md_funding +672行/45s 事件流正常，
  fin_opp_eval xvenue-carry 零行 → run.go:109 缺 `go` 实证。
- 首轮返修（9d1ce43）只给第二听加 go——阻塞的是第一听，第二行仍是
  死代码；部署后 LISTEN ev_funding 仍 2 条（research+dcd 臂）非 3 条，
  零日志零评估行抓出「修错行」。
- 全局扫：其余 ListenLoop 调用皆单听阻塞正解，仅 hypothesis 双听串行。
