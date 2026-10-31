---
schema_version: 2
id: checkall-parallel-load-flaky-timing
kind: pitfall
risk: medium
status: observed
keywords: [check_all 并行, flaky 测试, TestListenSocketStarvationRootFix, context canceled, CPU 争用, 绿戳]
scope: [internal/platform/events, scripts/check_all.sh, 绿戳并行盖戳面]
summary: 时敏测试在双 check_all 并行盖戳负载下 flaky（events 反压锚测 29s context canceled）——B 案 DSN 隔离只治 DB 争用，不治 CPU 争用；单测复跑绿=负载型 flaky，重跑补戳即可
first_seen: 2026-10-30
last_verified: 2026-10-30
skills: []
evidence_refs:
  - type: commit
    ref: dm/1030-closeout-docs 首轮 check_all FAIL=TestListenSocketStarvationRootFix（他席 5434 run 并行），单测 -count=1 复跑绿、二轮全链绿
  - type: commit
    ref: t/0268-首屏优化三件 同日复犯重症——5+ 席并行全谱下 5434 Migrate IO-wait 挂死→cmd/core/earn 双包 30min panic 假红；docker stats 择闲实例（o-postgres-test-b..e=5435-5438 池，-d/-e 实测零负载）专 DSN 重跑全绿
  - type: doc
    ref: internal/platform/events/events_test.go:147
  - type: doc
    ref: docs/tasks/T-0266-绿戳B案隔离库机化.md（B 案钦定界）
---

## 事实/症状

check_all 全谱 go test 段红：`TestListenSocketStarvationRootFix` 29.13s
`context canceled`（events_test.go:147「套接字未被排空→ClientWrite 钉死
实锤」）——docs-only 提交不可能破码面；同时段他席 check_all 在 5434
库并行跑（CPU 争用）。

## 根因

B 案（每席专库 DSN 隔离）治的是 **DB 层争用**（Migrate 锁/表锁假 FAIL）；
盒上 **CPU/调度争用**仍在——时敏测试（socket 反压/超时断言类）在双
全谱并行下超出其等待窗即假红。本类坑与「串行锁病灶」同症不同层。

## 处置

- 判 flaky 三步：红测试单跑 `-count=1`→绿=负载型 flaky 定性→全链
  check_all 重跑补戳（缓存命中下重跑成本低）。
- 并发挤兑窗先择闲实例：`docker stats o-postgres-test{,-b,-c,-d,-e}`
  （5434-5438 五实例池——默认 5434 全员挤兑），专 DSN 指零负载实例再跑；
  重症形=Migrate IO-wait 挂死→包级 30min panic（非真死锁）。
- 复发面：并行盖戳常态化（5435/5436 专库）后，events/时序断言族为
  首选嫌疑人——红先查是否并发跑窗内。
- 若同一测试连续两轮非负载环境仍红=真病灶，按常规根修径走。
