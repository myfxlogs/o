---
schema_version: 2
id: test-flood-exec-timeout-load-false-positive
kind: pitfall
risk: medium
status: verified
keywords: [check_all, flake, load, timeout, pg_notify, O_TEST_SHARDS, herd, 假阳性, GW-001, T-0468]
scope: [工程面, internal/platform/events, scripts]
summary: 洪泛式测试以「单发 Exec 超时」作服务端行为负断言时，墙钟阈按 CI 单机毛刺设计——分库 herd（O_TEST_SHARDS=16 同实例并发建库+迁移+测试）把服务端排队推过阈值=断言假阳性；判法=同测隔离绿全谱红，修法=阈按 herd 量级放大语义不动，勿把超时当可忽略放行
first_seen: 2026-11-08
last_seen: 2026-11-08
evidence_refs:
  - type: ticket
    ref: T-0468
  - type: ticket
    ref: GW-001
  - type: doc
    ref: internal/platform/events/events_test.go
---

## 坑面

`TestListenSocketStarvationRootFix` 洪泛件以单发 `pg_notify` Exec
限时 20s 作负断言（超时=服务端 ClientWrite 钉死实锤——正是被验的
排空语义的反面）。该阈按「CI 共享机排程毛刺」设计，O_TEST_SHARDS=16
分库后全谱跑=16 路并发建库+迁移+测试挤同一实例，服务端排队把单发
Exec 推过 20s——断言假阳性，check_all 两连红；同测隔离跑 K=1/K=16
六连绿=负载假阳性非语义破。

## 判据/修法

- 判法：同测隔离绿+全谱红=负载假阳性候选（先隔离对照再动阈）。
- 修法：容差阈按 herd 排队量级放大（20s→60s）判定语义不动——真
  钉死多少阈都红，负载型超时只放大阈不改断言。
- 勿采：把超时当「可忽略」放行——负断言正是被验对象本身。

## 证据

- check: check_all scoped 全谱两连红（events 包 FAIL）vs
  `O_TEST_SHARDS=16 go test -run TestListenSocketStarvationRootFix`
  隔离三连绿；
- result: 阈 20→60s 后单测绿（语义面未动）；
- ref: GW-001 段A 部署期 2026-11-08 实发。

## 补记（第二形态——竞态型）

同测试第三连红实证：floodStop() 取消 floodCtx 时若 Exec 恰在飞
（herd 下排队发>0 概率高），Exec 返回 context.Canceled 被误当反压
实锤——放大阈值治标不治本。修法：err 后查 floodCtx.Err()!=nil→
停灌拍取消在飞发=clean exit；自发 DeadlineExceeded 才是实锤。
判法增补：「停拍边界红」先查取消与在飞操作的竞态，再考虑放大阈。
