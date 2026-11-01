---
schema_version: 2
id: exec-state-literal-beyond-check
kind: pitfall
risk: high
status: verified
keywords: [fin_execution, CHECK 约束, 静默败, 幻影敞口, orphan, venue_err, 绑定参数未引用, dcdOpen, state 字面量]
scope: [cmd/core/open.go, fin_execution, o-core, PG CHECK 约束]
summary: state 字面量越 CHECK 集 + `_, _ =` 吞错 = UPDATE 永静默败、行留 'open' 幻影敞口喂 risk/earn 敞口聚合；同案伴生「参数绑了但 SQL 不引用」——venue 拒码看似落库实则蒸发。写面「形似有」比「没有」更毒。
first_seen: 2026-10-30
last_verified: 2026-10-30
skills: []
evidence_refs:
  - type: doc
    ref: migrations/fin_0082_exec_ambiguous_state.sql
  - type: doc
    ref: internal/risk/check.go:127
  - type: doc
    ref: internal/earn/worker.go:186
  - type: commit
    ref: T-0259 收口 commit（orphan→ambiguous 根修+venue_err_* 列实落）
---

## 事实/症状

T-0259 勘查 dcdOpen 拒因持久化面时发现两枚同型静默写坑：

1. `open.go` position 落位败径写 `fin_execution.state='orphan'`——0082 迁移的
   CHECK 集只有 `open/legged/rolled_back/closed/failed/rollback_failed/ambiguous`
   七值，'orphan' 从不在集：该 UPDATE **每一次都 CHECK 违例**，调用面 `_, _ =`
   吞错 → exec 行永留 'open' → risk.check/earn.worker 的 `state='open'` 敞口
   聚合吃进幻影在途额。
2. 同函 `dcd 业务拒` 径 `UPDATE ... SET state='failed' WHERE id=$2` 绑了
   `sCode+":"+sMsg` 作 `$1`——SQL 从不引用 `$1`：venue 拒码「绑了=看似持久化」
   实则蒸发，纯靠容器 stdout 留痕（10-29 容器重建毁证实证病灶本体）。

## 根因或策略解释

- 状态机 CHECK 集与写面字面量无交叉校验机——state 枚举扩集时旧写面静默
  脱节；`_ =` 吞错把唯一报警通道掐掉。
- PG 允许未引用绑定参数——「码传进 SQL」≠「码落列」，形似的持久化注释/
  参数比缺位更难被发现。

## 处置

- T-0259 同修：'orphan'→'ambiguous'（同语义合法态——exec 在仓行缺=查证面
  不吞）；失败标定统一走 `dcdMark/dcdMarkVE`——state+venue_err_code/msg
  同 UPDATE 原子写，写败 log ALERT 显形非吞。
- 凡写 state 字面量的径，对照迁移 CHECK 集核一眼；新增枚举态先查存量写面。

## 证据

- check: `grep -n "orphan" cmd/core/open.go` + `migrations/fin_0082` CHECK 集比对
- result: 'orphan' 不在七值集——该 UPDATE 恒违例；fin_0089 前 fin_execution 无
  venue_err_* 列，$1 绑定参数无落点
- ref: 本 commit（修后 `go test ./cmd/core -run TestDcdTradeRejectPersist` 绿）

## 反证条件

- check: `SELECT state FROM fin_execution WHERE state='orphan'`（任何库）
- falsifies_when: 存在 'orphan' 行=0082 CHECK 曾被绕过或本条目论断有误

## 适用边界

- 适用于一切「state 字面量 UPDATE + 吞错」写面；PG 绑定参数未引用不限于
  fin_execution——凡 SQL 参数位号与列引用脱节同病。
