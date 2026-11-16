---
schema_version: 2
id: schema-change-test-write-shape-replicas
kind: ops
risk: medium
status: observed
keywords: [迁移, 约束撤除, ON CONFLICT, 测试桩件, 写拍同形, 隐性消费面, fin_opp_eval, insert-only]
scope: [migrations, internal/**/*_test.go, 消费面审计]
summary: 撤 DB 约束/改写径语义时，测试件里「生产写拍同形」的 ON CONFLICT 辅助函数是隐性消费面——编译期零报错、运行期 42P10 才炸；消费面审计必须含 *_test.go 的 SQL 辅助件，且语义断言（同值静默/幂等重演）要随新写径重写而非只修语法。
first_seen: 2026-11-16
last_verified: 2026-11-16
skills: []
evidence_refs:
  - type: commit
    ref: c94b4b08d
  - type: doc
    ref: 工单#19778
---

## 事实/症状

工单#19778 撤 `fin_opp_eval` 自然键 UNIQUE 改版本化 insert-only。
check_all merge-scoped 测试红两处，皆 `SQLSTATE 42P10: there is no
unique or exclusion constraint matching the ON CONFLICT specification`：

- `internal/platform/events/notifyflood_test.go`——`upsertEval` 常量
  注释自承「生产写拍同形（funding 扫描臂 persist 同构）」，三处
  ON CONFLICT (inst_id,template,direction)；
- `internal/portfolio/lifecycle/machine_test.go:142`——mkProp 种子
  同形 upsert。

生产码 9 个写径已全剥 ON CONFLICT，唯独测试桩件的「同形复制品」
是审计盲区——grep 生产文件干净≠全仓干净。

## 根因

测试辅助 SQL 是**手工镜像的生产写拍形**，不随生产码自动演化；
撤约束后编译/静态检查全过（SQL 是字符串），只有真跑 PG 才炸。
更深一层：notifyflood 测试的语义断言（「同值覆写零 NOTIFY」「重放
=零醒」）建立在 upsert 覆写模型上——改成纯 INSERT 后同值重放
=新版本=**仍发通知**（每版皆新信息），断言须按新语义重写，
只删 ON CONFLICT 子句会让测试红在 expectQuiet 上。

## 处置

- 消费面审计清单加一条：`grep -rn 'ON CONFLICT' --include='*_test.go'`
  与生产写径同查；凡「X 同形/同构」注释的测试桩件=隐性消费面。
- 写径语义变更（upsert→insert-only/覆盖→追加）时，测试断言按
  **新语义重写**：静默面从「同值覆写」移到「UPDATE 同值臂」
  （uEval 镜像径仍守 IS DISTINCT 洪泛闸），INSERT 新版本=发。
- 洪泛量级复核：版通知=写率（~1-2/min）远低 T-0162 洪泛阈
  （~500/min），触发器函数本体不动——裁判面零改。

## 防复发

schema 变更工单的验收口径建议显式含「测试桩件同形面」——
本票若漏审，check_all 红在 merge-scoped 才兜住（幸运在闸有效）。
