---
schema_version: 2
id: testdb-residue-deterministic-red
kind: pitfall
risk: medium
status: observed
keywords: [check_all, go test, 测试库残留, o-postgres-test-c, 5436, TestGetCockpitHeadlessMerged, TestGetCockpitMissingDegrade, TestDcdOwnerDomainExcluded, No space left on device, ENOSPC, flaky 误判, testdb_lease]
scope: [o-postgres-test 族五实例池, scripts/testdb_lease.sh, scripts/check_all.sh, cmd/core, internal/risk]
summary: 「负载 flake」判词前先排「测试库残留确定性红」——o-postgres-test-c(5436) 残留态致 cockpit/dcd 三测确定性 FAIL（余四库全绿），换库 DSN 径即真绿戳；并发风暴叠加根分区 100% 另有瞬态 ENOSPC 红（cmd/core 迁移测试族）——两型皆环境病灶非码病
first_seen: 2026-11-08
last_verified: 2026-11-08
skills: []
evidence_refs:
  - type: commit
    ref: 外审 T-0450 裁——check_all 两跑于 5436 确定性红（TestGetCockpitHeadlessMerged/TestGetCockpitMissingDegrade/TestDcdOwnerDomainExcluded），同三测 5434/5435/5437/5438 四库全过、5436 独红两连；换 5437 DSN 全链绿戳@0ca0a554
  - type: commit
    ref: 外审 T-0454 裁——5435 跑红 cmd/core 四枚迁移测试=ERROR could not extend file No space left on device(SQLSTATE 53100)；同刻 relay 载 T-0460 工「host根分区100%打断check_all(go build link no space)」go clean -cache 释 130G 后复跑绿
  - type: doc
    ref: docs/tasks/T-0450-face-dep-pair-subreq-user-intent-trx-usdt-swap.md 收口段裁记断言③插曲
  - type: doc
    ref: scripts/testdb_lease.sh 外部 DSN 径（锁随端口走——择净库即绕残留）
---

## 事实/症状

check_all scoped 谱红 cmd/core `TestGetCockpitHeadlessMerged`/
`TestGetCockpitMissingDegrade`（「拔径未显缺 missing=[committed]」）
+risk `TestDcdOwnerDomainExcluded`（fail-closed 视图返 Allow）——
**同一测试在 o-postgres-test-c(5436) 上确定性红、在 5434/5435/5437/5438
上确定性绿**（单跑、整包、check_all 三粒度同结论）。非时序 flake：
两连复跑同红，换库即绿。

另一型：多席并发 check_all 风暴期根分区打 100%（go build link /
Go test 缓存 + PG 数据叠加），cmd/core 迁移测试族报
`SQLSTATE 53100 No space left on device`——瞬态盘满，释盘复跑即绿
（relay 同日 T-0460 工亲历在案）。

## 根因或策略解释

测试库非无菌——跑坏的测试/迁移可把状态残留在具体实例内，后续同库
跑的**状态敏感测试**（视图读/显缺面/fail-closed 判）被残留污染确定性
打死；DB 实例是变量，「同测试异库异果」即指纹。负载 flake 条目
（checkall-parallel-load-flaky-timing）治的是 CPU/时序层；本条=DB
内容层残留+磁盘容量层瞬爆——三层并存，判词不可混用。

## 处置

- 判红三辨（序）：①同测试异库异果→库残留（指纹=确定性+实例绑定）；
  ②`df -h /` 近 100% 或 SQLSTATE 53100→瞬态盘满；③时序断言超窗且
  双跑异果→真负载 flake。
- 库残留绕行=外部 DSN 择净库：`O_TEST_PG_DSN=postgres://o:o@localhost:
  <净端口>/o_test?sslmode=disable`（锁随端口走，testdb_lease 同源）。
- 盘满绕行=等并发退/`go clean -cache` 释缓存后复跑。
- 根治（库清残/容量线）非本条目域——立票面：5436 库清残或
  PG 数据目录容量阀入库治理。

## 证据

- check: `O_TEST_PG_DSN=…:5436 go test ./cmd/core -run TestGetCockpit*`
- result: 确定性 FAIL（两连）；同测 5437/5434/5435/5438 全 PASS
- ref: T-0450/T-0454 外审裁记（票面收口段）；relay 2026-11-08 12:28
  施工席 T-0460 异常行（根分区 100% 实证）

## 反证条件

- check: o-postgres-test-c 清残后同三测复跑
- falsifies_when: 5436 清残后测试转绿且他库无新残留——库残留说成立
  但「确定性」特征随库态漂移（残留键被后续跑清掉即转库）

## 适用边界

本机 o-postgres-test 族实例池（5434-5438 五实例+check_all 租约面）；
测试库治理（清残策略/容量阀）落位后条目转 retired 复核。外部 DSN
择库=人工绕行径，治证不治本——库残留属环境病灶，红案不可径记
「flaky」翻篇。
