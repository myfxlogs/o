---
schema_version: 2
id: seat-routing-worker-paused
kind: ops
risk: high
status: verified
keywords: [施工道停编, o-worker.timer, 席位路由, 外派施工, 工单桌无消费者, aseat, 完工回执, 施工收口, land_queue 自锁]
scope: [scripts/cognition/dispatch_worker.sh, scripts/land_queue.sh, ops_event, ops_ticket_desk, /opt/o-research/prompts]
summary: o-worker.timer 停编=有意停（无头不可见进度/卡死判），工单桌照发件但消费者死=管内票无声饿死（#29729 P1 逾期实证）——临时口径=新票默认外派（回执 note 必含完工回执，施工收口不咬 aseat 正则=自锁）/处置类归唤知/钱径令行值守直办；复席=业主令届时回收
first_seen: 2026-11-22
last_seen: 2026-11-22
evidence_refs:
  - type: ops_event
    ref: '#24535（RSI停席收口行）#29729（停册令 P1 逾期）#30603/#30626（evalfloor 自锁链）#30794（补回执解锁）'
  - type: ticket
    ref: 工单/landq-aseat-regex-poison（P1→O-27955 外派根修）
  - type: decision
    ref: 业主令 2026-11-22「先落，顺后交回无头」
---

## seat-routing-worker-paused —— 施工道停编期席位路由临时口径

### 病灶

o-worker.timer（施工席工单专办道，dispatch_worker.sh 拍）业主令停编——
**有意停非事故**：无头施工单元不可见进度、卡死不可判（2026-11-18 RSI
停席四 timer 中唤知/自我/审计三席已复位，独施工道留停）。但工单桌面
仍按「workable 票施工道自领」设计发件——**消费者死了件照立**，管内票
无声饿死（#29729 P1 逾期 23h+、#27955 P1 躺桌实证）。

### 临时路由口径（业主令 2026-11-22 钦定）

| 件类 | 归 | 形 |
|---|---|---|
| 新立工单（施工性质） | **默认外派** | note 载 `外派=施工`；档=/opt/o-research/prompts/O-<票>-<slug>.md；fwd: 转发词业主递外 |
| 处置类（销票/审批/判定/复审） | 唤知会话 | 双车道的信号道活着，照 playbook |
| 钱径令行（在册执行令——停册/参数行） | 值守/会话直办 | 非施工件，一道 UPDATE 级——#29729 值守直办先例 |
| 外派收口回执 | —— | note 必含「完工回执」+支名+40位全sha——land_queue review_ok 的 aseat 正则只咬（完工回执\|完工推支候审\|候审回执）词面，「施工收口」不咬=aseat 空 fail-closed 永不合（evalfloor 支实撞自锁） |

### 复席信号

无头施工道恢复=**业主令**（届时本口径回收，工单桌回自领制）。复席判据
候选：外派收口观察面成熟（进度可见+卡死可判）后业主裁。

### 关联

- ops_event #协议行（seat-routing 临时口径落账行 #30832）
- 工单 O-27955（aseat 引证污染根修——本口径第④条的码面根修票）
- 升件 #30757（ops_esc_pending 销件正则同脆——引证词面误销同类）
