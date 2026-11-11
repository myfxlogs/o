---
schema_version: 2
id: condition-gap-no-ops-event-pair
kind: pitfall
risk: medium
status: verified
keywords: [fin_condition_gap, 条件请求, ops_event, pg_notify, 事件驱动, LISTEN, 飞书桥, feishubridge, 轮询]
scope: [internal/platform/condreq, cmd/feishubridge, 事件源, 推送面]
summary: fin_condition_gap 请求行不配对 ops_event——写臂只落盒行不落事件行（hunger venue推进实证）；事件驱动消费面若认 ops_event 帧=丢件。正解=表级 pg_notify 触发器（fin_0164）+LISTEN 敲门，开机扫底追偿停机期
first_seen: 2026-11-11
last_verified: 2026-11-11
skills: []
evidence_refs:
  - type: doc
    ref: internal/portfolio/hunger/hunger.go:279
  - type: doc
    ref: internal/platform/condreq/condreq.go:14
  - type: server_fact
    source: postgres
    query: SELECT id,seat,event,note FROM ops_event WHERE event='条件请求' ORDER BY id DESC
    observed_at: 2026-11-11T13:00:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

ops_event 存在 event='条件请求' 行（#9278 处置席直写 stratretire 请托），
诱使消费面以为「条件请求=事件流成员」可读帧追件。实查写径：
condreq.Request（hunger venue推进 #279 调用点）只 INSERT fin_condition_gap，
**零 ops_event 落行**——两套「条件请求」是不同面：盒行=机器开口请求，
ops_event 行=席位直写请托文，无自动配对。

## 根因或策略解释

fin_condition_gap（D-0014 请求盒）与 ops_event（事实账本）是分立写面：
写臂各记各账，无触发器无桥接代码把盒行投成事件行。事件消费面
（relay urgent 曾查 `event='条件请求'`）只覆盖席位直写面——
盒写请求在事件流里隐形，帧唤消费架构对盒行=确定性丢件。

## 处置

fin_0164 表级事件源（本 commit 同入）：fin_condition_gap INSERT/UPDATE
至 pending→pg_notify('condreq_pending', id)——notify 只敲门不载正文，
读仍走表正源；o-feishu-bridge LISTEN 即醒推卡，开机扫底追偿停机期
漏件+水位文件跨重启不重推。D-0008-B 正解：零轮询。

## 证据

- check: cmd/feishubridge 三测试件+check_all 全链
- result: LISTEN 环替 30s 轮询过 D-0008-B 闸
- ref: migrations/fin_0164_condition_gap_notify.sql

## 反证条件

- check: 对比 24h 窗内 fin_condition_gap 新 pending 行数 vs
  ops_event event='条件请求' 行数——盒行远多于事件行即「不配对」仍真
- falsifies_when: condreq.Request 改为同写 ops_event（则表级 notify
  冗余，可降为帧唤消费）

## 适用边界

fin_condition_gap 消费面专用。ops_event event='条件请求' 行（席位直写
请托）是另一域——其推送仍走 relay urgent 文本径或后续另行裁定；
本桥件只拥盒行推送面。

## 伴生量表（T-0332 强制）

- gauge_name: condreq_stale_pending
- gauge_sql: SELECT count(*) FROM fin_condition_gap WHERE state='pending'
  AND requested_at<now()-interval '7 days'
- gauge_interval: 1d
- gauge_desc: 业主未答超 7d 的请求数——请求盒积压健康面（推送态在水位
  文件非 DB 面,桥件断流另由 o-feishu-bridge 单元活性监护）
