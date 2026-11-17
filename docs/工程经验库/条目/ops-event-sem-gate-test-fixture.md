---
id: ops-event-sem-gate-test-fixture
title: 测试往 ops_event 插行撞 (seat,event) 语义闸——矩阵外组合 P0001 loud 拒
kind: pit
date: 2026-11-18
source: 工单#23296 param_intake 桩测实证（param_intake_test.go 首跑）
---

# ops_event 写面语义闸对测试夹具同样生效（fin_0154→0189 矩阵累积）

**坑形**：桩测/脚本往 `ops_event` 直插行模拟事件史时，随手写的
`(seat,event)` 组合若是矩阵外对——BEFORE INSERT trigger
`ops_event_sem_gate` 直接 `RAISE EXCEPTION`（SQLSTATE P0001，
ERROR 明文 seat/event/ticket 三元）。测试炸在 fixture insert
而非被测逻辑，排错面偏离。

**实证**：#23296 进气道桩测自喂防行写 `seat='决策席-DM写径'
event='处置'`——`处置` 是闸内词（fin_0166/0187/0189 累积入册），
其 seat 白名单=处置席/复审席/施工席/研究席(源普查:%)/自我，
`决策席-DM写径` 不在列→拒写。真 DMSet 自产审计行走
`event='参数调整'`（矩阵外事件直通）故生产无此组合——
**测试夹具造了一个生产里物理不存在的行形**。

**判别**：fixture insert 报 `ops_event_sem_gate: 拒写非法
(seat,event)`——先查 `ops_event_sem_matrix` 在册对
（migrations/fin_0154+0159+0166+0187+0189 五迁累积），
闸内词（派单/裁决/驳回/领单/呈批/起草/拉起/已批/唤起/处置）
才查 seat 白名单；矩阵外 event 任意 seat 直通。

**修法**：模拟某写径的自产行=照该写径真实 (seat,event) 形
（DMSet=决策席-DM写径×参数调整）；模拟请求行=选矩阵内合法
seat（自我/唤知系×处置）或矩阵外 event。

**推广**：凡「直插业务表造前置态」的测试，先问该表有没有
BEFORE INSERT 语义闸——闸在库不在写径自觉（fin_0121 律），
裸 INSERT 一切写径闸全绕但库触发器一视同仁。
