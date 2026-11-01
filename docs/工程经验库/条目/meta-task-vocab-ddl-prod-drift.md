---
schema_version: 2
id: meta-task-vocab-ddl-prod-drift
kind: pitfall
risk: high
status: verified
keywords: [meta_task, task_transition, DDL漂移, 词表, 异常词, SG生产, 作废, 挂起, 候派]
scope: [meta仓, mbus-dispatch, SG o-postgres, meta.meta_task]
evidence:
  source: postgres
  query: SELECT word,ord,is_terminal FROM meta.task_phase_words ORDER BY ord
  observed_at: 2026-11-01T06:30:00Z
---

# meta_task 生产词表落后 repo DDL——作废/挂起/候派转相全落「异常词」

## 事实/症状

`meta.task_transition(t,'施工中','作废',…)` 返回 1 行（表象=转相成功），
实查行 phase=`异常词`、note=`非法词拒写落逃逸位：作废（原相=施工中）`。
函数按设计「非法词落异常词+RAISE WARNING 不静默吞」——psql 里 WARNING
只是 stderr 噪音，**count=1 把坠逃逸位伪装成成功**。

## 根因或策略解释

SG 生产 `meta.task_phase_words` 仅 13 词（DBP3a 初装集）；repo
`sql/meta_task.sql` 内 DBP3b 词表归一新增的 `作废/挂起/候派`（行 32-34）
+对应边集（行 59-70）从未部署到生产——repo DDL 与产线漂移。凡走缺词
转相（钦定裁死/停泊径）全坠 `异常词` 候裁，调用方看 RETURNING 行数
不可分辨。

## 处置

- 转相判定别看返回行数——`SELECT phase` 亲证目标相；`异常词→全相位`
  边可作绕行（实证：异常词→完成 三行成功落终）。
- 部署重放径钦定在脚本头：`ssh sg "docker exec -i o-postgres psql -U o
  -d o -v ON_ERROR_STOP=1" < sql/meta_task.sql`（幂等可重放）；重放前
  `SELECT` 存证词表/边集基线。
- 泛化：repo DDL 演进票验收锚必含「生产面 SELECT 实测=repo 声明」——
  影子写期「函数存在≠数据集齐」。

## 证据

- check: `SELECT word FROM meta.task_phase_words`（SG o-postgres）
- result: 13 词，缺 作废/挂起/候派；`meta_task.sql` 行 18-35 声明 16 词
- ref: obl372 清算实证 2026-11-01；T-0292 验收锚③

## 反证条件

- check: 生产词表 SELECT 含全部 16 词且边集=repo 全图
- falsifies_when: 部署重放已跑而词表仍缺词（=重放径本身失效，另立票）

## 适用边界

meta_task 相位转相面（dispatcher task_claim/report.sh/手工清算同受）；
词表补齐后本条目退役判据=系统已移除不适用——作 DDL 漂移类前车存查。
