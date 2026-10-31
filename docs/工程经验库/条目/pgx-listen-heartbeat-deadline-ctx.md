---
schema_version: 2
id: pgx-listen-heartbeat-deadline-ctx
kind: pitfall
risk: medium
status: verified
keywords: [pgx, WaitForNotification, LISTEN, 心跳, context deadline, Watch, SSE, DeadlineContextWatcherHandler, errTimeout]
scope: [internal/platform/events, cmd/core, pgx/v5, LISTEN/NOTIFY 长听径]
summary: pgx LISTEN 长听加应用层心跳的正解=逐迭代 ctx deadline——conn 在 deadline 后不杀（DeadlineContextWatcherHandler 只设/清 net.Conn 死线）；errTimeout unwrap 到 context.DeadlineExceeded 可判；select 于 ticker+WaitForNotification 不可行（conn 非 goroutine 安全）
first_seen: 2026-10-30
last_verified: 2026-10-30
skills: []
evidence_refs:
  - type: doc
    ref: cmd/core/service.go:182-234
  - type: doc
    ref: cmd/core/watch_hb_test.go
  - type: commit
    ref: T-0269
---

## 事实/症状

pgx `Conn.WaitForNotification(ctx)` 阻塞等 NOTIFY——要给长听流（SSE/Connect
server-streaming）加应用层心跳帧时，直觉解 select 于 ticker+读 goroutine
不可行：**同一 *pgconn.PgConn 不可并发用**（WaitForNotification 独占读套接字）。

## 根因或策略解释

pgx v5 默认 `BuildContextWatcherHandler = DeadlineContextWatcherHandler`：
ctx 到期→只 `Conn.SetDeadline(now)`（读失败返回 timeout），`Unwatch`→清死线——
**连接不关闭可复用**。返回错误=`errTimeout{err: ctx.Err()}`（有 Unwrap——
`errors.Is(err, context.DeadlineExceeded)` 成立）。`CancelRequestContextWatcherHandler`
才是杀连径（非默认）。

## 处置

正解=逐迭代 `context.WithTimeout` 包住 WaitForNotification（期界等待形，
避开 NewTicker 周期原语门禁）：

```go
for {
    wctx, cancel := context.WithTimeout(ctx, hbInterval)
    n, err := conn.Conn().WaitForNotification(wctx)
    cancel()
    if n != nil { /* 转发事件帧（n/err 可同时非 nil——缓冲通知先派） */ }
    if err == nil { continue }
    if !errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil { return err }
    /* 发心跳帧 */
}
```

判序要点：`n != nil` 先派事件再判 err（`pgx.Conn` 层缓冲通知与错误同返
不丢帧）；父 ctx 死/真断连照返不重试。

## 证据

- check: `go test ./cmd/core -run TestWatchHeartbeatFrames -v`（期界缩 60ms
  var 缝——产态 15s）
- result: PASS——watch_open 后静默窗 watch_hb×2 按拍到，ev_account NOTIFY
  透传不饿死；favicon 挂接同测过
- ref: T-0269 收口 commit

## 反证条件

- check: 升级 pgx 主版本后复跑同测 + `grep BuildContextWatcherHandler pgconn/config.go`
- falsifies_when: 默认 handler 改为杀连形（deadline 后 conn 不可用→每拍重连）

## 适用边界

pgx v5.11（本仓 go.mod 钉版）；DeadlineContextWatcherHandler 默认装配前提
下成立——自配 BuildContextWatcherHandler 的径不适用。
