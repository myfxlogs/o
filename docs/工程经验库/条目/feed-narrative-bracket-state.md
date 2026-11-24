---
schema_version: 2
id: feed-narrative-bracket-state
kind: pitfall
risk: medium
status: verified
keywords: [feed_push, 叙事聚合, unit锚, 会话窗, fin_push_ledger, 水位, 双渲染, 支名提取, charset]
scope: [呈现层, O·动推送, ops_event读面]
summary: 事件聚合成叙事的窗态必须持久化（重启不丢抑制行）+闭标三分流（feed可见交行序解/剔集库查补解/缺失超时兜底）+先账后推；支名提取须 charset 限定否则 \\S+ 吞并后随 jargon
first_seen: 2026-11-24
last_verified: 2026-11-24
skills: [施工席]
evidence_refs:
  - type: doc
    ref: cmd/core/feed_narrative.go（feedNarrState/narrativeBlocks 三段式——PhaseA库查补解+超时兜底 / PhaseB行序开闭窗）
  - type: doc
    ref: cmd/core/feed_push.go pending()（feedRow.ts 增列——叙事「活了多久」要素）
  - type: doc
    ref: cmd/core/feed_human.go feedHumanBranchRe（支=(\\S+)→支=([A-Za-z0-9_./-]+)——实证吞「——唤知复审席三维签」）
  - type: server_fact
    ref: 生产 ops_event 24h unit= 分布——在办199/205、会话收口150/163、认领14/14 有锚；处置/复审过/完工回执/落地 0 锚（窗员全靠时段归属）
---

## 事实/症状

O-36963 业主令「O·动要故事线非流水账」：同 unit 会话（在办→会话收口）
四行 18 秒「醒→签复审→销报警→收口」被逐行推成四条互不解的断句。
聚合层施工三处结构坑：

1. **窗态不持久化=重启丢行**：会话窗内成员行被「抑制待叙事」，若窗态
   只存内存，core 重启后水位已过这些行 id——永不再渲染=静默丢行。
   修法=窗态落 fin_push_ledger（scope=sys_status 复用推送水位表），
   读写与水位同生命周期。
2. **闭标三分流不可一刀切**：会话闭标=会话收口（feed 可见）∨唤起果/
   唤起败（feedExclude 剔集永不入 pending）∨缺失（会话暴毙）。
   若 Phase A 库查把会话收口也预解→该行后续入批无窗可归→叙事+交班行
   双渲且叙事锚错位；若唤起果/唤起败不库查→此二闭的会话窗永挂起。
   修法=库查只判「首闭是不是剔集闭」——会话收口首闭留 Phase B 行序解。
3. **先账后推**：窗态先于推送文落库——发败窗态仍真（下拍续跟/兜底），
   存败不发（下拍全量重试）。反序=发成存败→已闭窗复活重叙=双推。

## 判据

- 叙事=加成非替换：无 unit 锚/窗外交行/聚合败全退现行逐行渲染，
  信息面零消灭；窗员按 id 区间库查回取（不依赖 pending 重取——
  水位可越过抑制行，叙事不受水位截断）。
- 超时兜底只兜「永不闭」：有闭标（哪怕未入批）不走超时——龄再长
  闭标在库行到即叙事。
- 窗内系统事件（升件/判词/部署）不吸：要事不延迟，feedPierce 语义不破。

## 坑源（支名吞并）

`支=(\S+)` 把「支=fix/x——唤知复审席三维签」整段吞进支名——非空白
分隔的中文注记全被吸。支名词表本就 charset 闭集
（fix|feat|canon|audit|t|revfix|hotfix + [\w./-]），提取必须同限——
`\S+` 在含全角破折注记的 note 上必然越界。教训=「截到空白」不是
「截到词界」，机产 note 的 k=v 值后常跟无空格注记。
