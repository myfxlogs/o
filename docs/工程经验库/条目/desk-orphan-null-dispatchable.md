---
id: desk-orphan-null-dispatchable
title: 三值逻辑 NULL 陷阱——NOT dispatchable 对 rank NULL 票判 NULL，积压计数含尸失效
kind: pit
date: 2026-11-24
source: 工单O-35463（21 张 rank NULL 无主票三重隐身实证）
---

# 票桌无主死角（dispatchable NULL 三值陷阱+分诊无器官）

**坑形**：`ops_ticket_desk.dispatchable = workable AND rank<=wip`——
排位集 `w` 只收「workable∧证=锚∧末act≠receipt」票，无证/回执剔票
`dispatch_rank=NULL`→`rank<=N` 判 NULL→`dispatchable=NULL`（非 false）。
`WHERE workable AND NOT dispatchable` 三值逻辑下 NULL 行被滤出——
**计数含尸失效**：无主票对积压计数隐身（dispatch_worker BACKLOG_N
与 ops_state 工单面压力臂 bl 双双漏计）。

**连环死角**（单点修不全的三环）：
1. fin_0212 证锚闸把无证票挡出派发池=设计本意（逼分诊），但分诊
   职责无器官承接→只进不出；
2. 自域票（feed-*/landq-*/session-*/queue-* 等）撞工人剔名单∧唤知
   票道只剩 rulable+receipt→两边都不接；
3. `NOT dispatchable` NULL 判→积压压力卡都不计，三重隐身。

**修法**：谓词一律 `dispatchable IS NOT TRUE`（NULL→true 显式含尸，
fin_0239 迁移修 ops_state 活体+dispatch_worker.sh BACKLOG_N 同式）；
孤儿池压力卡=探测细胞聚合升件（workable∧rank NULL∧龄>24h，计数+
最陈 id+死因粗分，池清零自销）挂 hungerTick 同槽。

**判词**：凡「X IS NOT TRUE」优于「NOT X」当列可空——NULL 是第三
态不是假；排查「计数对不上」先查谓词对 NULL 的三值语义。
清册归零三径=close（交付/过时/dup）/补证（新票带证=锚接续）/
挂起（条件+期标挂由）；goal-* 票=锚件非派发件，rank NULL 系票型
天然，销=杀目标锚（hunger_gap 抑键同殁）——挂起留锚不销。
