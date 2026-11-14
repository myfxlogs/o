---
schema_version: 2
id: eval-belief-overwrite-era-mismatch
kind: pitfall
risk: high
status: verified
keywords: [fin_opp_eval, upsert覆写, 入场信念, 时代错配, calib_drift,
  失真幻影, funding-hedge, era_skip, ON CONFLICT DO UPDATE]
scope: [eval 行语义(活页vs化石), 预测-兑现对账面, 信念可证伪性]
summary: fin_opp_eval 以(inst,template,direction)键 upsert 原地覆写
  ——行内容=乐器当前信念活页，非开户时信念化石。calib_drift 探针
  JOIN 提案 eval_id 拿到的可能是开户后重估值，「预测-兑现失真」
  实为时代错配幻影。funding-hedge 实证：负簇 eval.ts 均开户后18天，
  「预测-14.4% vs 实测+7.5%」全非判官判反（工单#15866）。修=
  探针时代守卫(e.ts<=pr.created_at)；根修(eval史/提案快照)
  归工单续案。
first_seen: 2026-11-14
last_verified: 2026-11-14
skills: []
evidence_refs:
  - type: doc
    ref: internal/research/grammar/evalwrite.go upsert ON CONFLICT (inst_id,template,direction)
  - type: doc
    ref: internal/research/funding/persist.go verdict 原地改 rejected
  - type: doc
    ref: cmd/core/worldmodel_esc.go calibDriftEscCheck 时代守卫(de21f0627)
  - type: ticket
    ref: 工单#15866 eval-calib-funding-hedge
  - type: server_fact
    source: postgres
    query: SELECT avg(EXTRACT(EPOCH FROM(e.ts-s.created_at))/86400) FROM fin_opp_eval e JOIN fin_proposal p ON p.eval_id=e.id JOIN fin_strategy s ON s.proposal_ref=p.id WHERE s.template_ref='funding-hedge' AND e.net_apr<-0.1
    observed_at: 2026-11-14T11:40:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

`calib_drift` 探针报 funding-hedge 族「预测 -14.4% vs 实测 +7.5%」
（工单#15866/独白#16130「符号都反了」）。取证三连：
- 预测双峰非单向偏：505 eval 判<-10% vs 1088 判正——均值是簇合成；
- 负簇 eval.ts 平均开户**后** 17.9 天（min 5.9d），正簇开户前 1.7d
  ——负簇行是开户后重估，非入场信念；
- verdict 谱矛盾：rejected 判下竟有 retired/live 后代——verdict
  同样被事后改写（persist.go 起草闸原地 UPDATE）。

## 根因

`fin_opp_eval` 表 `(inst_id,template,direction)` 唯一键 + upsert
`DO UPDATE SET ts=now(),net_apr=...,verdict=...`——行是「该乐器
此刻的信念」活页，每次重估覆写销毁历史。而 calib 对账的可证伪
对象应是「开户时刻的信念」——提案 eval_id 永远指向最新覆写值，
入场信念无化石可寻，且无 ops_event 镜像。拿事后悲观重估对真实
兑现=幻影失真：判官没错，对账单时代错了。

## 修法/判据

①探针时代守卫（本 commit de21f0627）：仅 `e.ts<=pr.created_at`
的入场时代信念入 n/pred/real；改写户计 `era_skip` 随升件如实报数
（`tpl(预测x% vs 实测y%,n=N,改写M)`）——不可对账探沉默=如实。
②根修归工单续案（eval 信念史）：方向二选一——eval 行版化
（insert-only 新行，提案钉入场版 id）或提案链接时快照
（inst,apr,verdict,ts 入 fin_proposal 列）。属 research/grammar
业务径语义，归唤知会话判词施工，外席不越权。
③凡「预测-兑现」类对账面，先问「被比对象是可覆写活页还是入场
化石」——upsert 键上的信念行永远不可直接当历史证据。

## 通用化

「册载信念行」与「事件账事实行」要分得清：可覆写行只代表此刻，
ops_event 才代表历史。任何取证/对账/审计面 JOIN 业务活页表当
历史用=同款时代错配坑在等。判据写法：凡对账单，先证比对两侧
时间轴同一锚点（此处=开户时刻）再谈偏差大小。
