---
schema_version: 2
id: listen-loops-starve-pgxpool
kind: pitfall
risk: high
status: verified
keywords: [pgxpool, LISTEN/NOTIFY, 连接池饿死, 事件脊, 静默断流, watchdog 盲区, MaxConns]
scope: [internal/store/pgstore, internal/platform/events, 全域 LISTEN 消费者, 部署面]
summary: 每路 LISTEN/NOTIFY 永久占一条池连接——事件脊扩到 7 路即吃光 pgxpool 默认容量（max(4,NumCPU)），采集写/巡扫全部饿死且零日志
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: []
evidence_refs:
  - type: server_fact
    ref: SG md_ticks 自 08:59 新镜像部署起断流 40min；容器 Recv-Q 21-91KB 积压（WS 帧到达未读）；池仅 4 established 全被 LISTEN 占
---

# LISTEN 长连接吃光 pgxpool 默认容量→采集饿死静默断流（T-0050 实勘）

## 病灶
- pgxpool.New 裸用=MaxConns max(4,NumCPU)；T-0048 事件脊 7 路 ListenLoop 各占一条永久连接→池满→okxws 批量写 SendBatch 永久等 acquire→行队列(1024)塞满→读循环堵在 queue 发送→Recv-Q 积压。
- **静默三重奏**：写败零日志（计数器有账无出口）、读堵、okxws watchdog `lastFrame!=0` 前置条件在「帧从未到达写路径」时恒 skip——三个监控面同时失明。

## 裁决
- 长连接消费者（LISTEN/每连接会话）上线前**必算池容量账**：MaxConns ≥ Σ长连接+峰值短查询；显式 NewWithConfig 禁裸 New。
- watchdog 判死条件禁用「起表值非零」类前置——订阅成功即起表，无帧=从 0 起判。
- 采集断流排查序：先 netns Recv-Q（帧到没到/读没读）再查写径——Recv-Q 积压=读侧堵，DB 空转≠venue 断。

## 判据
- 新增 ListenLoop/常驻连接面时 review 必查 pool 配置；「subscribed 后零数据零错误日志」=本病灶体征。
