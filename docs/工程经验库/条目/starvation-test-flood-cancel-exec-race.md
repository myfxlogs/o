---
schema_version: 2
id: starvation-test-flood-cancel-exec-race
kind: pitfall
risk: medium
status: observed
keywords: [TestListenSocketStarvationRootFix, flake, CI, context canceled, pg_notify, 竞态, 桩测]
scope: [internal/platform/events, CI ubuntu-latest, 桩测时序]
summary: "TestListenSocketStarvationRootFix 灌装 goroutine 停灌 cancel 撞上在途 Exec→「通知发射被反压 context canceled」假红——CI 共享机排程毛刺实证（同包本地 51.9s 绿、CI 12:53 红一次），修径=Exec 败先判 floodCtx.Err() 分流收摊 vs 真反压"
first_seen: 2026-10-28
last_verified: 2026-10-28
skills: []
evidence_refs:
  - type: code_fact
    ref: internal/platform/events/events_test.go:88-94——floodStop() cancel floodCtx 时在途 notifier.Exec 返回 context canceled 被记成 floodErr 反压实锤
  - type: provenance
    ref: CI run 40690936661 build-test 红（events 包 40.174s FAIL）；同 SHA 本地 check_all events 51.929s ok——环境时序 flake
---

## 事实/症状

`go test ./internal/platform/events` 在 CI ubuntu-latest 偶发：
`events_test.go:147: 通知发射被反压（套接字未被排空→服务端交付 ClientWrite 钉死实锤）: context canceled`——FAIL 后整包红。同 SHA 本地全谱绿。

## 根因

灌装 goroutine 的 `notifier.Exec(ectx, pg_notify...)`（20s 超时容差）在
`t.Fatal` 主径 `floodStop()` 撤 floodCtx 的瞬间仍在途——ectx 为 floodCtx
子代被取消，Exec 返回 `context canceled`，被错误径记成「反压实锤」
`floodErr`，主径 `t.Fatal(<-floodErr)` 误报。竞态窗=一次 Exec 耗时内；
CI 共享机排程毛刺放大窗口（本地近零概率）。

语义上 cancel 撞在途=**停灌收摊正常形**，非反压——桩测把收摊竞态误判
成病灶，自报假红。

## 处置

修径（另单，施工票边界外未动）：Exec 败先分流——
`if err != nil { if floodCtx.Err() != nil { floodErr <- nil; return } ... }`
（floodCtx 已撤=收摊形静默离场，仅真超时/真反压记 floodErr）。

## 证据

- check: CI run 40690936661（T-0219 PR#14 首跑）build-test 红——events 包仅此一测败
- result: 本地 check_all 同包 51.929s ok 全绿；本地连续跑未复现
- ref: `gh run view 40690936661 --log-failed`（2026-10-28T12:53:02Z）

## 反证条件

- check: 修后 CI 连续 N 跑零 flake
- falsifies_when: 分流守卫加入后仍报反压——则非收摊竞态而是真交付卡死，本条退役重查

## 适用边界

- 仅该桩测灌装 goroutine 收摊序；生产 listenOnce 路径无此竞态（生产 fn 长拍不灌自杀）
