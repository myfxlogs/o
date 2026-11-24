---
id: supervisor-kids-residual-never-respawn
title: Supervisor kids 残行——goroutine 死 map 占位=永不 respawn
kind: pit
date: 2026-11-25
source: 工单 O-38013（cex-supervisor-respawn——okxpriv 凭据缺件户僵尸态实证）
---

# kids 残行病灶（okxpriv Supervisor 实证）

**坑形**：名册驱动 Supervisor（syncRoster diff 起停子会话）里，子
goroutine（acctLoop）在 spawn 前置段失败早退（凭据缺件=readCreds
败），但 `kids[venue]=cancel` 占位行**无人清**——下拍 syncRoster
diff 恒见「该户在册」跳过 spawn，凭据补齐后也永远不活。表现=
roster state 停 degraded、venue 数据面静默断流，名册看起来「在册
在跑」实际零会话。

**判别**：roster 行 state=degraded ∧ last_error=凭据读败 ∧
会话面无帧——查 Supervisor 内部 map 才见 goroutine 已死占位仍在。
对拍法=「map 在册 ⇔ goroutine 活」不变式破缺。

**修法**：早退路径自清——`noteSpawnMiss` 内 `cancel()+delete(kids[v])`
（cancel 幂等自收无害）+退避账（credMiss 计数/retryAt 指数退避
2s→60s 顶格按拍续试不放弃——凭据=外部供物迟到即活）；syncRoster
spawn 环加 `time.Now().Before(retryAt[v])` 闸；出 want 的 venue
连退避账一并清（disabled/removed 永不复活由 cexAccounts state 过滤
构造性保证）。spawn 成功径 `spawnOK` 清零退避账。

**推广**：凡「map 登记=任务在跑」的 supervisor/worker 池，任务体
的**所有**早退路径都必须回收登记行或标 dead 可重 spawn 态——
「起成功退失败」不对称是僵尸态温床。同型面=mt5 Collector runs
map（同构风险点，spawn 前置段若增可败径须同律）。
