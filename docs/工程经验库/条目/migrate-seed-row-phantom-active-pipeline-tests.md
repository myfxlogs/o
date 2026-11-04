---
schema_version: 2
id: migrate-seed-row-phantom-active-pipeline-tests
kind: pitfall
risk: medium
status: verified
keywords: [迁移种子行, ops_event, fin_0097, fin_0099, 活跃票+1, TestGetPipeline, TestMigrateChainSurface, 断言漂移]
scope: [test, migrations, pipeline_face, fixture]
summary: 迁移件 INSERT 业务种子行（fin_0097 ops_event T-0298'启用'）与新建桥表（fin_0099 fin_opp_hypothesis）会让「断言计数/缺席」类测试集体假红——fixture 预期须随链面种子同步，写种子/新表的迁移须同 commit 对账受影响断言
first_seen: 2026-11-04
last_seen: 2026-11-04
evidence_refs:
  - type: doc
    ref: migrations/fin_0097_basis_keys.sql:11（INSERT ops_event T-0298'启用'种子）
  - type: doc
    ref: migrations/fin_0099_ghost_bridge.sql:23（fin 链建 fin_opp_hypothesis）
  - type: doc
    ref: cmd/core/pipeline_face_test.go:135/196/229（expected 2/22/2 实得 3/23/3）
  - type: doc
    ref: internal/store/pgstore/migrate_test.go:91（fin 链不应有 fin_opp_hypothesis 断言）
---

# 迁移种子行/新建表→测试断言集体假红（T-0318 裁决实证）

## 现象

`O_TEST_PG_DSN` 下 `go test ./...` master 确定性红两组（无 DSN=全 skip
假绿——testutil-dsn-unset-skip-false-green 同族遮罩）：

- `TestGetPipelineEndToEnd/Truncation/PhantomRowNormalized`：
  活跃票断言恒 **+1**（expect 2/22/2 → got 3/23/3）。
- `TestMigrateChainSurface/fin`：「fin 链不应有 md 域表
  fin_opp_hypothesis」。

## 根因（双件同族——断言漂移非码红）

- fin_0097 迁移 `INSERT INTO ops_event('施工席','启用','T-0298')`——
  每个新迁 schema 自带一张「启用」末事件活跃票（无完成/裁决/收口谓词
  →恒 active），活跃票计数类断言集体 +1。
- fin_0099 迁移在 fin 链 `CREATE TABLE fin_opp_hypothesis`（domain.tsv
  已归 bridge 域），ChainSurface fin 段的「不应有」断言陈旧未同步。

与共享库残行无关：o_test public.ops_event 残行仅 T-9999（active=1），
测试 schema 隔离本在——红源=**迁移册自身种子/表集 vs 断言册漂移**。

## 教训

- 迁移件往**判据型基表**（ops_event/ops_headless 等读径重建源）写种子行
  =隐式改全测试基线；表集增删同理。「断言计数/缺席」测试=迁移面隐式
  下游，加种子/新表迁移须同 commit 跑受影响断言对账。
- 判红先二分：`master 同诊`+`diff 零触码/迁`→不罚本件立追修；勿把
  断言漂移算到收口件头上。
