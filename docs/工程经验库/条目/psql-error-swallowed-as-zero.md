---
schema_version: 2
id: psql-error-swallowed-as-zero
kind: pitfall
risk: high
status: verified
keywords: [psql, 语法错, 静默零值, fail-open, dispatch_worker, TICKET_N, 孤儿括号, 拾桌谓词, 唤起死, SQL字面量, 2>/dev/null, 默认值兜底]
scope: [scripts/cognition, scripts 计数查询面, ops_event 唤起契约]
summary: psql 查询错被 ${VAR:-0} 吞成「零候选」=fail-open死症——597b90b94 拾桌谓词孤儿右括号致 dispatch_worker TICKET_N 恒空、唤起静默死五日（backlog 9→32 实证）；修=语法锚+桩测病灶回归锚
first_seen: 2026-11-23
last_verified: 2026-11-23
skills: [施工席]
evidence_refs:
  - type: doc
    ref: ops_event#（O-34419 收口回执——病灶实证与修复锚）
---

## 现象

`dispatch_worker.sh` 拾桌计数查询自 2026-11-18（597b90b94 引入
剔除谓词时）带孤儿右括号 `feed-' )`——psql 报 syntax error，
`TICKET_N=$(psql ... 2>/dev/null)` 得空串，`TICKET_N=${TICKET_N:-0}`
兜底成 0 → `[ "$TICKET_N" -ge 1 ] || exit 0` 每拍静默退出。
ops_event「唤起」行 11-18 后绝迹，工单 backlog 9→32 淤积——
派发器死了五天无人知。

## 根因或策略解释

三层叠加缺陷：

1. **SQL 字面量孤儿括号**：`note !~ '^...feed-' )`——引号内正则
   收 `feed-`，其后 ` )` 无配对 `(`，整句 syntax error。写端笔误，
   无人跑真库故五日未爆。
2. **错误吞为零值**：`VAR=$(cmd 2>/dev/null); VAR=${VAR:-0}` 模式
   把「查询失败」与「真零行」压成同一值——fail-open 方向恰好是
   最坏方向（该唤不唤）。计数兜底语义只对「键缺席」成立，
   对「语法错/连接断」必须分诊。
3. **静默退出无告警**：exit 0 + 无 note/心跳差分——死件与闲件
   在账面同形（这正是 resident/自观面要补的盲区）。

## 处置

- 修=删孤儿 `)`（dispatch_worker.sh + 同谓词面 batchdesk.sh 同改），
  桩测加病灶锚 `! grep -q "feed-' )"` 双文件回归钉。
- 判据沉淀：**计数兜底只兜「缺席」不兜「错」**——`VAR=$(psql)` 后
  应判空串与合法 0 分流（空=查询败→告警径；0=真空窗→正常退避）。
- 同族面：凡 `2>/dev/null` + `${VAR:-默认}` 组合的拾桌/计数查询，
  SQL 语法面改动必须真库跑通一遍（桩 psql 不验语法）。
