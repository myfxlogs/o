---
schema_version: 2
id: dual-listenloop-serial-starvation
kind: pitfall
risk: high
status: verified
keywords: [ListenLoop, 阻塞死循环, 串行调用, 监听臂零跑, ev_funding, WaitForNotification]
scope: [internal/platform/events, internal/research/hypothesis, 事件驱动臂]
summary: ListenLoop 阻塞死循环——Run 内顺序调两次则第二听永不到行（research-hypo-sig 自 T-0140 零跑实证）；「无产出」与「无卡如实」表象全同，验收须先证臂在跑
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
  第二及以后调用必须 `go`。
- 验证事件链三层分验：trigger 在库 ≠ NOTIFY 发出 ≠ 监听臂跑到 fn。
- 「无卡=正确」类验收须先证臂在跑（心跳/计数器/日志痕），缺席证据≠阴性。

## 实证
- T-0144 验收窗（2026-10-26）：md_funding +672行/45s 事件流正常，
  fin_opp_eval xvenue-carry 零行 → run.go:109 缺 `go` 实证。
- 全局扫：其余 ListenLoop 调用皆单听阻塞正解，仅 hypothesis 双听串行。
