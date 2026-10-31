---
schema_version: 2
id: test-schema-public-fallback-poison
kind: pitfall
risk: medium
status: verified
keywords: [测试基建, testutil, schema 隔离, search_path, public 残表, DROP TABLE CASCADE, 拔径桩, 共享实例]
scope: [internal/platform/testutil, go test 拔径桩写法, 共享 O_TEST_PG_DSN 实例]
summary: testutil.Pool 每测独立 schema（t_<pkg>_<nanos>）但 AfterConnect 注入的 search_path 恒带 public 兜底——共享测试实例 public schema 若有历史残表（实证 o_test public.ops_event 10-29 残数据），unqualified 表名在「本 schema 已拆表」后解析兜到 public 残表=拔径测试假活；拔径桩必须 ①qualified DROP（SELECT current_schema() 拼名，禁裸 DROP TABLE）+ ②独立池钉 search_path 去 public（SET search_path TO <schema> 无 public）
first_seen: 2026-10-31
last_verified: 2026-10-31
skills: [生产取证]
evidence_refs:
  - type: server_fact
    source: postgres
    ref: o_test(5434) public.ops_event 存 10-29 残行（T-9999 ts=2026-10-29）；TestGetCockpitPipelineMerged 裸 DROP 后 GetPipeline 仍返残表数据假活
  - type: code
    ref: cmd/core/pipeline_face_test.go TestGetCockpitPipelineMerged（qualified DROP+隔离池修后真显缺）
  - type: code
    ref: internal/platform/testutil/testutil.go AfterConnect "SET search_path TO schema,public"（兜底源）
---

# 共享测试实例 public 残表兜底——拔径桩假活

## 症状

拔径测试（DROP TABLE 后断言查询显缺/显错）绿不了：`DROP TABLE ops_event CASCADE` 无报错执行完，被测查询却照常返回**陈旧数据**（行 ts 竟是两天前），missing 名段不出。

## 根因

testutil.Pool 的 schema 隔离是**加法非减法**：每测新 schema 建在共享实例上，`AfterConnect SET search_path TO t_<pkg>_<nanos>,public`——**public 永远兜底**。共享实例 public schema 一旦被历史运行（schema 隔离上线前/CI 老链）留下同名残表，两个解析口径就分叉：

- **读**：本 schema 无表 → search_path 顺位兜到 public 残表 → 查询「假活」返回残数据；
- **写**：裸 `DROP TABLE <名>` 同样 unqualified → 解析到 public 残表（误伤共享面）或本 schema 表，两可。

## 对策（写拔径桩的固定两步）

1. **qualified 拆表**：先 `SELECT current_schema()` 取本测 schema 名，拼 qualified `DROP TABLE <schema>.<表> CASCADE`——绝不裸拆；
2. **去兜底验尸**：拔径断言前用**新池**（AfterConnect `SET search_path TO <schema>`，无 public）发查询——解析无处兜底必败，missing/err 断言才真。

## 回改点

- cmd/core/pipeline_face_test.go TestGetCockpitPipelineMerged（两步全用——本坑首修现场）。
