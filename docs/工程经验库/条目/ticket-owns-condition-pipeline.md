---
schema_version: 2
id: ticket-owns-condition-pipeline
kind: design
risk: high
status: verified
keywords: [工单, 票抑, scope, 认领, 挂起条件, 流水线, 查分派修审上线, escInsert, dispatch]
scope: [cmd/core/feed_esc.go, scripts/cognition/dispatch.sh, scripts/cognition/runbook.md, ops_event]
summary: 「查分派修审上线」接续环=工单一等票型三态推导:OPEN票触发唤知(派)+抑同scope升件(管催);挂票期限内仍抑不触发;销/过期=抑撤探针复岗。认领TTL=2h防双会话抢票。探针重火与票管催分流=RSI空转(同判词N销)根修。
first_seen: 2026-11-11
---

# 票管催制——工单三态推导与探针抑火（查分派修审上线接续环）

**设计契约（2026-11-11 业主令「修好RSI让他无障跑」立）**：

票态全从 ops_event 推导（append-only 律,无 UPDATE）：

- **OPEN**：无处置行引「工单#id」→
  ① dispatch 触发面在案即唤知（派站接续——不等龄,票面即优先级）；
  ② escInsert 抑同 `scope=` 的探针升件（条件被票接管,
  不再重火问同一问题——`commercial_stall` 同值29销/
  `live_dud` N销那类空转断根）。
- **SUSPENDED**：处置引 id 且载「挂起条件=X」→不触发唤知
  （票开着但不该干）,但期限内仍抑升件（等外因≠没人管）。
- **CLOSED**：处置引 id 无挂起标记**且非接续行**→哨撤,条件
  还在探针自然复岗（escalateUnresolved 未销才抑=抑撤即重火）。
  终局判据剔双标（审计#12643 实证,详 ticket-close-finality-
  predicate）：`挂起条件=`行非终局（限内暂抑过期回哨）；
  `工单#id ?接续`引证行非终局（进度报告）——剔接续必须锚
  到票引证,裸 `%接续%` 会误剔真销行（#11287 附言含
  「接续件」案）。

**认领**：`event='认领'` 载「工单#id」——TTL=2h 对齐
RuntimeMaxSec；活在认领防双会话抢同票,认领陈=会话死透
票回落可拣（票不死,只是换手）。

**挂起标记判据**：`挂起条件=`——非 `挂起` 裸词（销案词
「非挂起」含「挂起」子串曾致销票误判成挂票,抑不撤）。

**探针面归一**：escEdge 沿触发版+escalate 同窗 dedup 版
删除——全探针归一 escEdgeU/escalateUnresolved（销了重火
是票抑复岗前提;沿触发版死沿被抑消费=票一抑 scope 永哑）。

**复审三层**（机审不装能审一切）：L1 判据可执行化
（票面写明测试名,修序=先挂测→修→测绿）；L2 审计臂
`销案复审` 域（测试真编码判据否——同义反复=伪证）；
L3 票涉资金上限/风控参数/凭据/迁移约/append-only 表
→条件请求上行业主。
