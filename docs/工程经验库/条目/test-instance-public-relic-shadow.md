---
schema_version: 2
id: test-instance-public-relic-shadow
kind: pitfall
risk: high
status: verified
keywords: [测试实例, public schema, relic, 遮蔽, search_path, class 列缺失, 静默零值, CREATE IF NOT EXISTS]
scope: [o-postgres-test 主实例, testutil.Pool, schema 隔离, 迁移 CREATE IF NOT EXISTS]
summary: 主测试实例 public 层 T-0021 前 relic 表遮蔽 per-schema 建表——缺列查询被吞错 helper 静默变 0；施工检面换干净备席容器
first_seen: 2026-11-01
last_verified: 2026-11-01
skills: []
evidence_refs:
  - type: server_fact
    ref: o-postgres-test public.fin_cashflow 无 class 列；TestWeeklyAnnualized 主实例 ann=0、备席 -e 同测试绿
  - type: doc
    ref: internal/platform/testutil/testutil.go (search_path TO schema,public)
---

# 测试实例 public relic 遮蔽 schema 隔离（T-0252 施工发现）

## 现象

o-postgres-test（主实例 :5434）public schema 存 T-0021 schema 隔离立例前的
relic 表（`public.fin_cashflow` 缺 fin_0015+ 后续列——无 class 列、旧 kind
闭集）。测试经 testutil `SET search_path TO t_<pkg>_<rand>,public` 后，迁移
`CREATE TABLE IF NOT EXISTS fin_cashflow` 解析到 public 已存在→跳过建——
所有新 schema 的 fin_cashflow 实为 public relic。

## 判据（钉死）

- class 过滤查询（如 report net=Σincome `WHERE class='income'`）在 relic 上
  `column "class" does not exist`——被 q() 吞错 helper 静默变 0：测试假绿/
  断言零值难分辨。
- 验别式：`SELECT count(*) FROM information_schema.tables WHERE
  table_schema='public' AND table_name='fin_cashflow'` >0 = relic 在。
- 复现：主实例 TestWeeklyAnnualized ann=0 而 base 正常；干净备席
  （-b..-e :5435-5438）同测试绿。

## 处置

- 施工席：检面换干净备席——`O_TEST_PG_DSN=postgres://o:o@localhost:5438/
  o_test?sslmode=disable`（-e；-b..-d 同净）。零碰共享实例。
- 运维待办（决策/管理席）：主实例 public relic 择空窗 DROP（24 活连接在飞
  时勿动）；gc_test_schemas.sh 只清 t_* schema 层，不覆盖 public。

## 关联

T-0021 schema 隔离、T-0186 schema GC、T-0266 多实例、T-0252 施工实证。
