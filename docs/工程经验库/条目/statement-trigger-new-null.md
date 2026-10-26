---
schema_version: 2
id: statement-trigger-new-null
kind: pitfall
risk: medium
status: verified
keywords: [trigger, STATEMENT, NEW NULL, 行级, 幂等, 回扫, plpgsql]
scope: [migrations, DB trigger, 回扫/审计面]
summary: plpgsql trigger 函数引用 NEW 行语义必须 FOR EACH ROW——STATEMENT 级 trigger 的 NEW=NULL，WHERE inst_id=NEW.inst_id 零匹配零报错（静默不触发回扫，INSERT 照常成功=假活面）；行级 trigger + PK/状态谓词幂等，batch 入册逐行等价（T-0147 实案）
first_seen: 2026-10-26
last_verified: 2026-10-26
skills: [策略评估与仓位]
evidence_refs:
  - type: code_fact
    ref: migrations/0061——STATEMENT 级首版手工复现（INSERT 成功 state 不变），ROW 级改后桩测绿；ztrace rowCount 诊断法（GET DIAGNOSTICS ROW_COUNT 入表）
  - type: doc
    ref: docs/tasks/T-0147（结果回填——入册钩回扫 trigger 幂等二重语义）
---

# STATEMENT 级 trigger 的 NEW=NULL=静默零匹配假活面

## 病灶形

0061 回扫 trigger 首版 `FOR EACH STATEMENT`——函数体 `UPDATE ... WHERE
inst_id=NEW.inst_id`：STATEMENT 级 trigger **NEW 为 NULL**→谓词 `=NULL`
恒零行→INSERT 成功、回扫静默不触发（无报错=假活面）。行级语义写在
STATEMENT 级 trigger 里=典型错配。

## 判据挂接

- trigger 函数引用 NEW/OLD→必须 `FOR EACH ROW`；STATEMENT 级只做集合级
  全表操作（禁引用行值）。
- trigger 桩测必须**端到端断言行效果**（state 翻转/行数差分），非只看
  INSERT 成功——静默零匹配=INSERT 照常成功，无错误信号。
- 诊断法：GET DIAGNOSTICS rc=ROW_COUNT 落 trace 表——rowCount=0 即解析/
  匹配层断，一眼定位。

## 处置

T-0147：0061 改 FOR EACH ROW+TestRetroactiveScanTrigger 端到端锚
（state 翻转+transition 溯源+幂等恒 1 行）。
