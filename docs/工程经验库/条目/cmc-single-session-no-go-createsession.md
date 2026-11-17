---
schema_version: 2
id: cmc-single-session-no-go-createsession
kind: pitfall
risk: high
status: verified
keywords: [cmc, 单会话, createsession, 会话互踢, 无头浏览器, AuthManager, 爬虫]
scope: [cmc, auth, cmcweb]
summary: CMC 个人盘单会话制——任何 Go 侧 createsession/refresh 重登立即踢掉浏览器 web 会话（实证乒乓 23:04/23:15 双杀）；修法=Go 侧零会话，浏览器 UI 登录=唯一入口，REST 全走页内 fetch 代理（令牌源=CDP 旁录 app WS 发帧 auth 真 JWT）
first_seen: 2026-11-16
last_seen: 2026-11-16
evidence_refs:
  - type: log
    ref: "cmcweb 23:04:15 app WS closed（o-core refreshJWT 500→重登同秒）；重载落登录页=会话被杀非轮转；重登后 23:15:01 再杀（cmcweb AuthManager 23:12 重登）"
  - type: commit
    ref: "e142d7ca8 探测:cmcweb单会话制根修"
---
## 现象
浏览器 UI 登录+点行+app WS 建立全通，但 WS 活几分钟后关闭且重载落登录页——不是 app 内轮转，是会话被外部 createsession 杀死。时间线对拍：o-core 与 cmcweb 各自的 AuthManager refresh 500→重登→踢掉浏览器会话，浏览器再登录又反踢 Go 侧，无限乒乓。

## 根因
CMC 个人盘服务端单会话制：同一账户同时只允许一个 web 会话，新 createsession 使旧会话 token 失效。任何保活 Go 侧 AuthManager（哪怕只为 REST 轮询）都与浏览器会话互斥。

## 修法（钉死）
- `cmcweb` 内禁出现 `cmc.AuthManager`/`Login`/`StartRefresh`——UI 登录=唯一合法入口。
- `o-core` 中继径（`cmc.ws_base` 或 `cmc.rest_base` 覆写任一存在即触发）用 `StaticJWT` 零会话；直模式（无覆写）仍走 AuthManager 原径。
- REST 需求全走 cmcweb `/api/` 页内 fetch 代理：令牌=CDP `EventWebSocketFrameSent` 旁录 app 自发 auth 帧里的真 JWT，与 app 请求同栈同指纹。
- 中继 `serveWS` 上游死=503 拒升——陈旧回放假报活闸。
