---
schema_version: 2
id: standing-condition-as-event-churn
kind: pitfall
risk: high
status: verified
keywords: [常亮告警, 升件, 边缘触发, 值指纹, escEdgeUV, commercial_stall, idle_capital, live_dud, ops_event, 积压]
scope: [cmd/core, 探测薄壳, 升件面, escLoop]
summary: 常设态信号被当事件建模——销案后同值每拍复火填塞升件面（commercial_stall 同值第 35 火）；根修=值指纹边缘触发，同值已阅不火、异值重火、销票破纹保复岗网、30d 记忆期定期点名
first_seen: 2026-11-11
last_verified: 2026-11-11
skills: []
evidence_refs:
  - type: commit
    ref: 7c5c18349
  - type: doc
    ref: cmd/core/feed_esc.go:184
  - type: server_fact
    source: postgres
    query: SELECT esc_scope,count(*) FROM ops_event WHERE event='升件' GROUP BY 1 HAVING count(*)>10 ORDER BY 2 DESC
    observed_at: 2026-11-11T04:30:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

commercial_stall 同量值（-2.53）第 35 火、idle_capital/live_dud 逐拍复火——
升件面被「同判词延续」销案 churn 占满，真新件（账链断 #10143）混沉其中。
业主令：「常亮告警是常量，放积压里不对」——状态被当事件建模。

## 根因或策略解释

escalateUnresolved 的 dedup 粒=「未决事件」：销案即让位，条件仍真→下一拍重写
新行。对**事件**（deploy 败/断流新批）此语义正确——每次发生=新事证；对
**常设态/计量面**（利用率、净产出、哑弹数、积压数、二元死态）错误——同量值
复火不是新事证，是同一现实的重述。「销案≠条件消失」与「复火=新事证」两条
钦定语义在计量面上自相矛盾，判词 churn 是矛盾的账面显形。

## 处置

escEdgeUV 值指纹边缘触发（commit 7c5c18349，部署 o-core:7c5c18349814）：
- 指纹载 note 前缀 `v=<fp>`（不载 ticket——ticket=工单流分组键,face 面有消费）
- 本 scope 30d 内末行（开或销）纹同=同值已阅不火；纹异=新现实重火；
  30d 记忆期=同值超龄定期点名防静默烂账
- 销票破纹：末行升件后本 scope 票已终（非挂起销∨挂票过期）=指纹失效照火
  ——销票复岗安全网不破（TestTicketScopeSuppress 契约）
- 三纹型：计数纹（n）、水位纹（util%/net/模板集/ids集）、死态回合纹 `@`
  =deadSince 起刻（同回合销后不火,复绿再死=新回合照火）
- 16 枚条件探针全转；escEdgeU(fp="") 留作事件型默认口——新探针须显式选型
- 已销升件复绿不出恢复行：销案行即关闭语义,无未销可引

## 证据

- check: check_all scoped go test（TestEscValueFingerprint+TestTicketScopeSuppress 四分支）
- result: 全链绿+绿戳+rc=0 部署回执在案
- ref: commit 7c5c18349；/opt/o/.git/check-green-7c5c18349*

## 反证条件

- check: 观察销案后 30d 内同 scope 同 note 前缀升件行数>1
- falsifies_when: 同值复火再现=纹闸失效或被绕（查 escEdgeU 裸调新点）

## 适用边界

探测薄壳升件面。事件型探针（每次发生=新事证）仍走 escEdgeU 空纹旧语义。
`@` 回合纹记进程内存——o-core 重启=新回合纹,同态销后复火一发（fail-open
取向,兼作重启点名）。计数纹对同计数换成员钝感（8 枚换 1 枚纹同——量化代价，
值变类纹无此限）。复火簇残余监护=伴生量表周期读（不立 meta 针——机制已吸
收不变式,双写规则+误报即成所治之病）。

## 伴生量表（T-0332 强制）

- gauge_name: esc_refire_cluster
- class: 探针健康
- query: SELECT esc_scope,count(*) FROM ops_event WHERE event='升件' AND ts>now()-interval '7 days' GROUP BY 1 HAVING count(*)>3
- threshold: >3 同 scope 7d 复火=指纹闸失效/新探针误挂事件型——人工复核
