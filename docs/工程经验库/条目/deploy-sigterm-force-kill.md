---
schema_version: 2
id: deploy-sigterm-force-kill
kind: pitfall
risk: medium
status: observed
keywords: [SIGTERM, 强杀, failed to exit within 10s, 优雅退场, docker stop, 部署窗, o-core, signal.NotifyContext]
scope: [部署径, cmd/core, 容器生命周期]
summary: o-core 收 SIGTERM 后 10s 内不退出——docker 每次 stop/重建都走「failed to exit within 10s — using the force」SIGKILL 强杀（10-27 以来 journal 144 次）。优雅退场路径（signal.NotifyContext→run 返回→pool.Close）实际从未在部署窗内走完过。
first_seen: 2026-10-31
last_verified: 2026-11-07
skills: []
evidence_refs:
  - type: provenance
    ref: SG journalctl docker[1051] "Container failed to exit within 10s of signal 15 - using the force"（10-31 11:22 deploy 53b2a3cf01cb 窗实拍；2026-11-07 统计 144 次）
  - type: doc
    ref: cmd/core/main.go:54（signal.NotifyContext 装配在案——退场链某环 >10s 悬置）
---

## 事实/症状

SG journal：每次 o-core 容器 stop/重建，dockerd 记 "stopping restart-manager" → 10s 后 "Container failed to exit within 10s of signal 15 - using the force"（SIGKILL）。10-27 以来 144 次（含 alpha 等他容器——o-core 10-31 11:22 与 11-07 02:12 两窗亲证）。dev 栈同码 o-core 容器 Exited(0) 在案（10-29 14:35 起 107s 寿命）——退出码 0 说明最终退场链可达，只是 >10s。

## 根因或策略解释

signal.NotifyContext 装配在案（main.go:54），ctx 取消后 run() 应返回——但 10s 窗内没走完。候选悬置环（未逐环实测，本条只记观察）：gRPC/http srv.Shutdown、pgxpool.Close（MaxConns=20 池全收）、roster/collector 等 goroutine 阻塞 defer 链。后果：每次部署=SIGKILL 中飞——在飞写截断、优雅钩子不跑、 bounded-state upsert 靠幂等兜底（现无实证损害，属脆弱面非现行病灶）。

## 处置

**未修**（T-0434 范围外——只记坑）。候选修向：退场链逐环计时 log（srv.Shutdown 带 ctx timeout、pool.Close 前置超时）；验证后再定是否需要激进退场（10s 内强制 return）。留常票候选。

## 证据

- check: `ssh SG "journalctl -u docker --since 2026-10-27 | grep -c 'failed to exit within 10s'"` → 144
- result: 强杀=部署窗常态，非偶发

## 反证条件

- falsifies_when: 部署窗 journal 出现 o-core 10s 内干净退场（无 force 行）——退场链已被修

## 适用边界

只覆盖 docker 径部署；systemd 面（OCI 二进制件）退场语义另判。
