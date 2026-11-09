---
schema_version: 2
id: mt5-gateway-addr-startup-cache
kind: pitfall
risk: medium
status: verified
keywords: [mt5, gateway_addr, collector, 启动缓存, 端点切换不生效, OpenSession]
scope: [internal/mt5, app_config]
summary: collector NewCollector 启动时读一次 mt5.gateway_addr 存 c.addr，长会话重连环永不重读——换端点须重启进程才生效（OpenSession 每拍重读无此坑）。
first_seen: 2026-11-09
last_verified: 2026-11-09
---

mt5 网关 addr 启动缓存——换端点须重启才生效

**坑**：`internal/mt5/collector.go` `NewCollector` 启动时读一次
`mt5.gateway_addr` 存 `c.addr`，长会话重连环 `dial(c.addr)` 永不重读
config——改 app_config 端点后长会话仍打旧址直到进程重启。
`OpenSession`（session.go）每拍重读无此坑——同仓两径行为不对称。

**实证**：2026-11-09 grpc3→grpc2 切换后，OpenSession 立即走新址 authed，
collector 长会话仍 60s 一拍 `会话终止` 打 grpc3；`docker restart o-core`
后 md_ticks mt5 行恢复写入（ops_event #8931）。

**修法候选**：重连环每轮 readCreds/读 addr（同 OpenSession 径）；
或 config.Watch 代次变化时刷 c.addr。
