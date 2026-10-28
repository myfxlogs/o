---
schema_version: 2
id: test-gate-hold-conn-fatal-cleanup-hang
kind: pitfall
risk: high
status: verified
keywords: [go test, t.Fatal, pool.Close, 挂死, 超时, gate channel, 测试夹具, CI 超时]
scope: [test, events, pgxpool, cleanup]
summary: 测试里阻塞型 fn 攥池连接+gate 只在成功路径放行=任意 t.Fatal 早退→pool.Close() 永等还连→整包挂死 30min，真败因被吞
first_seen: 2026-10-28
last_verified: 2026-10-28
skills: []
evidence_refs:
  - type: server_fact
    ref: GitHub Actions run 40647880430 panic stack（puddle Pool.Close WaitGroup.Wait + fn 停 events_test.go:38 gate）
---

## 阻塞型夹具件攥池资源+t.Fatal 早退=清理环挂死吞真败因

- **实证**：`TestListenSocketStarvationRootFix` 的 fn 首拍停 `<-gate`
  （攥 LISTEN 池连接），`close(gate)` 只在第 143 行成功路径——CI 上
  flood 发射超阈触发 `t.Fatal(:134)` → Goexit → defer 链跑
  `pool.Close()` 永等未还连接 → 整包 30min panic，CI 三连（真败因
  反压告警文本从未落日志，只见挂死栈）。
- **教训**：测试里的「放行闸」必须挂在 defer/ctx 双径——阻塞型回调
  攥着池连接/锁/事务时，只走成功路径的放行=败例路径结构性泄漏；
  `pool.Close()`/`wg.Wait` 会把泄漏放大成整包超时。
- **判据**：测试写「chan 放行型夹具」三查——①gate 有 defer 放行兜底
  ②阻塞等待 select ctx.Done 自解径 ③t.Fatal 全路径手排一遍（每条
  Fatalf 后资源还回不回得去）。
