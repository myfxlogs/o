---
schema_version: 2
id: ws-session-jwt-bound-relogin-storm
kind: pitfall
risk: medium
status: verified
keywords: [websocket, jwt, 会话续期, cmc, 重登风暴, 爬虫, auth帧]
scope: [cmc, ws, auth]
summary: WS 服务端会话寿命=挂载 auth-JWT 寿命（ttl300s 到期掐线）——拿断线重连+全量重登兜底=~288次/日登录风暴脚印；修=连内续期 auth 帧（浏览器同款），服务端不认则原重连径兜底零回退
first_seen: 2026-11-16
last_seen: 2026-11-16
evidence_refs:
  - type: log
    ref: "o-core [cmc-ws] session expired → full re-login ok×14/6h"
---

## 现象

CMC WS 采集腿每 ~5min 被服务端 "Session expired" 掐线（1011），代码把断线
当会话死→每次 4 步全量重登（createsession→SIGN_IN→SILENT→JWT）：6h=14 次
全链登录≈288 次/日，账户面脚印吵如攻击者。会话本身能自愈但传输层在烂。

## 根因

WS 连接的服务端会话寿命=挂载它的 auth JWT 寿命（CMC JWT ttl=300s，exp 实采）。
代码只在建连时取一次 JWT，连接存活期从不续期→JWT 一到期服务端掐线。
下游再聪明的"过期即重登"补丁（D-370 maybeRelogin）都是治标——
正确问题是**怎么不死**而不是**死后怎么起**。

## 修法

连内续期：JWT 到期前 ~45s（`reauthWait` 按 exp 重排，贴线钳 30s 下限）在
同一连接重发 `{"auth":{"token":新JWT}}`——浏览器客户端同款行为。
服务端不认=照旧掐线由既有重连径兜底，纯加法零回退。
同类正例：mt5.session_token 共享行（单 login 属主+复挂永不 ConnectEx）——
"少登录、多续命"是 venue 会话面通用律。

## 免疫件

- `internal/data/cmc/ws_reauth.go` startReauth goroutine + `reauthWait` 时序闸
- `ws_reauth_test.go`：300s 新鲜/贴线/过期/非法串四档时序断言
- 产线观测面：`[cmc-ws] auth ack status=` 日志（续期回应帧）+ `连内续期 auth 帧已发`
