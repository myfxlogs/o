---
id: rebase-must-reuse-recon-same-source-rows
title: 一次性重锚/校准件必须复用对拍同源行集装配——第二套净额数学=必漂移
kind: pit
date: 2026-11-25
source: 工单O-33206 ledger-rebase（业主令账本重锚）
---

# 重锚净额数学同源律

**坑形**：给对账域做「以 venue 实读为新基线」的一次性重锚/校准时，
若另写一套净额计算（cashflow 求和±自判组件归户），与周期对账的
recon 净额（cashflow+parked/dcdBase/nativeBase/mtm 归户+读面存在性
过滤）必然漂移——重锚「归零」的只是自己那套数学，探针下拍照报。

**判别**：重锚/校准类簿记件的验收锚=「探针残差→0」——实现后拿对拍
行集逐键核 net==eq，而不是只核自己算的 delta 行数对。

**修法**：把对拍行集装配抽成共享函数（O-33206=`reconRows`——
reconAccounts 判定段与 rebaseDecree 重锚段同吃一份），重锚只消费
行集差值 delta=eq−net 落 recon_adjust；前置闸（在途窗/postInterest/
venue 读面）与对账同序同件（`inflightActive` 抽出共用）。

**推广**：凡「人工/一次性校准 vs 周期对账」双数学并存域，校准件
禁独立重算——要么共享行集，要么校准后由对账实证归零才算完。
幂等=桶级源名查重+armed→done 单次旗；静态豁免退役限「差不会再生」
型（resolver 跟踪型构性漂移豁免必须保留，否则下拍误报）。
