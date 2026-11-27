---
id: markpx-readsite-semantics-drift
title: mark/latest 价读件语义拷贝漂移——同族表达式 5 份平行实现各自缺件
kind: pit
date: 2026-11-27
source: 工单 O-41385（业主令持仓盈亏卡——「第 5 个读者不得再造字面拷贝」钦定根修）
---

# mark 价源读件语义拷贝漂移（O-41382/41384 后遗症）

**坑形**：O-41382（报价行 last 恒 0→双边中价）与 O-41384（ticks 降级订阅
面会死→md_mark_index 双源 union）两轮根修后，正确价源形
`COALESCE(NULLIF(last,0),(NULLIF(bid,0)+NULLIF(ask,0))/2)` ∪
`md_mark_index channel='mark-price'` 在仓内长出 **5 份语义拷贝**——
dirpaper markQ / dirtrig lastPx / dirline mark 子查询 / notionalNow /
sizing LatestPx。拷贝间已实证漂移：dirtrig/notionalNow 的 mark_index 支
**漏 channel 过滤**（md_mark_index 双 channel 实证：mark-price 545 行 +
index-tickers 502 行——index 价≠mark 价，串源=尸价喂判官）；dirpaper 是
分支独占（SWAP→mark_index ONLY）非 union；sizing LatestPx mark_index 支
连 channel/venue 过滤全无。每加一个读者就多一份漂移面，第 6 个拷贝
（盈亏卡）被业主当场叫停。

**判别**：同一「最新可代价」语义在多文件以 SQL 字面量复刻——grep
`NULLIF(last`/`(bid+ask)/2` 出现处 >1 即病灶；逐份 diff 匹配谓词
（venue/inst/channel 过滤面）必有出入。

**修法**：`legset.MarkPx(ctx,pool,venue,inst)` 单源——中价式+双源 union
逐字钦定形；**鲜度界判权归调用点**（各点 maxAge/cfdMaxAge 语义原样保留，
禁把界判收进价源件）；返回 px+markTS+src 胜出源（src 供 detail 审计位）。
venue="" = inst_id 直配+venue<>'sim'（读点现行匹配面）；venue≠"" =
双源同加 venue=$1（dirline 形，sim 演习腿照原样可达——venue=$1 已收窄
域，不叠加 venue<>'sim'）。

**推广**：价源/判据类 SQL 表达式凡被第二个消费点引用=该收编共享件的
信号；「只换价源表达式不改匹配语义」迁移纪律——各点 inst 匹配方言
（venue 谓词有无/sim 过滤有无/channel 过滤）逐字保留，收编时顺手补的
过滤（mark_index channel='mark-price'）属钦定形一部分非私货。验收 grep
`NULLIF(last`/`(bid+ask)/2` internal/ 域 ≤ legset 单件+测试；残余
`(bid+ask)/2` 命中（slipSample 需 bid/ask/last 分量/xvenue·xshadow·mt5
bridge 简单中价）是**异语义读点**非本族拷贝——判别看是否含
`NULLIF(last,0)` 末价优先形，裸 `(bid+ask)/2` 是另一表达式。
