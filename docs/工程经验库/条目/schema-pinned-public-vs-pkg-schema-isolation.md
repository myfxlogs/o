---
schema_version: 2
id: schema-pinned-public-vs-pkg-schema-isolation
kind: pitfall
risk: medium
status: verified
keywords: [schema, search_path, current_schema, testutil, T-0021, fail-closed, 迁移播种]
scope: [retention, migrations, testutil, pgstore]
summary: 新 SQL/迁移面钉死 schemaname='public' 会与 testutil 包级 schema 隔离（T-0021）互斥——测试 schema 下播种/守门/统计全空转，且 fail-closed 路径表现为「静默全拒动」而非报错
first_seen: 2026-10-26
last_seen: 2026-10-26
evidence_refs:
  - type: provenance
    ref: T-0153 CI 红（TestSweepDrivenDelete 删 0 留 2，c6f896c 起 26 连红）
---

## schema 钉 public × 包级 schema 隔离=静默空转

- **实证**：T-0153 清道夫三处钉 public——0063 播种 `pg_tables WHERE schemaname='public'`、reportUnregistered `table_schema='public'`、sizeSentinel `"public."+t`。testutil 池 `search_path=t_<pkg>_<rand>,public`，迁移表全落测试 schema → 播种 0 行 → `fin_data_retention` 空账 → 清道夫 fail-closed「未裁窗=拒动」逐表 skip → 清窗 0 行。CI 红 26 连（13:05 起全 T-0153 后 commit）。prod 面（迁移落 public）完全无症状——病灶只在测试坐标显形。
- **教训**：①新写 SQL/迁移引用 schema 时一律 search_path 感知（`current_schema()`/`ANY(current_schemas(false))`/`to_regclass(current_schema()||'.'||$1)`），勿字面钉 'public'；②fail-closed 路径的失败形态是**静默拒动**不是报错——被测行为依赖账/册/配置行存在时，先断言「行在」，否则测的是 skip 径不是工作径；③本地曾报绿的验收桩若 DSN 缺席走 skip 语义，skip≠过（本次 0.005s「绿」实为全 skip）。
- **收口实证**：三处改 search_path 感知后，0063 播种 `fin_forward_check→audit_keep/30d` 在案，retention 3 桩绿，全量 go test 39 包绿。
