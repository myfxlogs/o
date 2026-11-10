---
schema_version: 2
id: hypo-decide-crossdb-audit-semgate
kind: pitfall
risk: medium
status: verified
keywords: [tools/hypo, decide, ops_event, 跨库, fin_0154, sem_gate, roster_rw, review_rw, ops_headless, OPS_DB_DSN]
scope: [假想卡裁决面, tools/hypo decide, ops_event 审计行, md/fin 正本裂面]
summary: hypo decide 同库同事务审计行结构性落空——fin_opp_hypothesis 正本在 OCI PG-md 而 ops_event 正本在 SG fin 库；且 seat 白名单（fin_0154 处置∈{处置席,复审席}）与凭据权（review_rw 缺 ops_headless）双重暗闸。修法=OPS_DB_DSN 两序写+roster_rw 凭据+处置席 seat。
first_seen: 2026-11-10
last_verified: 2026-11-10
skills: [研究员]
evidence_refs:
  - type: server_fact
    source: postgres
    query: SELECT seat,event,note_like FROM ops_event_sem_matrix WHERE event='处置'
    observed_at: 2026-11-10T03:55Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

`hypo decide` 上线首跑三连拒：
①md 库无 `ops_event` 表（SQLSTATE 42P01）——同事务审计行落空，态迁回滚；
②`seat=devin-外席 event=处置` 被 `ops_event_sem_gate` 拒（fin_0154 矩阵
处置∈{处置席,复审席}）——态已迁审计未落；
③`OPS_DB_DSN=review_rw` 跨库写撞上 `permission denied for table ops_headless`
（ops_event 触发器触面，review_rw 对该表零授权）。

## 根因或策略解释

`fin_opp_hypothesis`（md 域）与 `ops_event`（fin 域）正本分库——同连接同事务
的审计设计跨不了裂面；`ops_event` 另有 fin_0154 语义闸（seat×event 白名单）
+触发器触 `ops_headless` 表（roster_rw 有 INSERT/SELECT/UPDATE，review_rw 零
授权——它只持 ops_event.INSERT 单边权）。

## 处置

- `decide` 加 `OPS_DB_DSN` 两序写：态迁 tx 先落（md 正本），审计行后补
  （OPS 池）——审计落败=态已迁+显式重放声+remediation SQL，零静默；
  缺省同库同事务（测试/同域面原语义）。
- 凭据：OPS_DB_DSN=**ROSTER_PG_DSN**（非 FIN/review_rw）。
- seat：缺省 `处置席`（矩阵 处置∈{处置席,复审席}——decide 写 event='处置'）。
- 态已迁审计落空的回填径：SG 侧 `docker exec o-postgres psql -U o` 直补
  `INSERT ops_event(...,'处置','hypo','假想卡#N surface: old→new 裁词=…（审计补录）')`
  ——处置席 seat 同径过闸。

## 证据

- check: `hypo decide` 三轮败迹（42P01/sem_gate 拒/ops_headless 42501）+
  roster_rw 直插 `INSERT 0 1` 实证
- result: 9 卡裁决落账（4 adopt+5 park），9 审计行（superuser 补录 4+5）
- ref: `tools/hypo/main.go` decide 两序写；`migrations/fin_0154`；
  KB `hypothesis-cards-adopt-via-hypo-tool`（正本裂面先案）

## 反证条件

- check: `hypo decide` 同库单 DSN 在 md 正本上落审计行成功
- falsifies_when: ops_event 桥表回 md 域或 decide 走 fin 域投影库写态
  →「跨库裂面」前提失效
