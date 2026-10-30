---
schema_version: 2
id: h1-origin-h2-mismatch-diagnosis-path
kind: method
risk: low
status: verified
keywords: [HTTP2, ERR_HTTP2_PING_FAILED, Cloudflare, cloudflared tunnel, h1-only origin, 会话级连坐, favicon 同挂]
scope: [cmd/core/main.go, cmd/core/service.go, web/src/react/stream.ts, 链路诊断面]
summary: origin 纯 HTTP/1.1 架构下前端报 HTTP/2 故障——h2 只存在于浏览器↔CF 边缘一段；排查径=逐段协议钉死+prior-knowledge 证伪探针，误去服务端装 http2.Server 是无腿可配的错配方向
first_seen: 2026-10-30
last_verified: 2026-10-30
source: 诊断-前端HTTP2事故-1030.md（管理令 MANAGE-401③ 钦定条目化）
---

## 事实/症状

业主浏览器报障：全 RPC+favicon 同秒 ERR_HTTP2_PING_FAILED、状态栏
却 200 OK。直觉方向是「服务端 h2 keepalive 未配」——但 origin 根本
没有 h2 腿：`main.go` 裸 `http.Server` 无 TLS/h2c，`curl
--http2-prior-knowledge` 打 50051 直接被拒（h2 证伪实锤）。

## 根因（排查径=本条目核心）

链路 HTTP/2 **只有一段**：浏览器↔Cloudflare 边缘。`PING_FAILED` 是
Chrome 在该段 PING 无 PONG 后撕毁整条 spdy session 的会话级错误——
天然解释「全部复用请求连坐+响应头已收 body 断」签名。逐段钉死法：

1. `curl -sI https://域名/` 看 `HTTP/2`+`alt-svc`——边缘段协议实证；
2. SG `ss -tlnp`+`ps`——无反代，`cloudflared --protocol http2
   --token-file` 4×HA 连接=CF↔origin 段；
3. `curl --http2-prior-knowledge` 打 origin 端口——被拒即 h1-only
   证伪（免翻配置猜）；
4. tunnel syslog/metrics 查断连史——边缘↔tunnel 腿健康度；
5. 残余唯一未测段=浏览器↔CF 边缘（跨境长连接被中间盒黑洞化，无
   RST→PING 打空）——钉死需业主侧 net-export/换网/禁 QUIC 三选一。

干净路径 e2e 阴性对照（五页零失败+流 150s 全存活）同样是证词——
把「路径病」与「码病」分辨开。

## 处置

- 服务端 `http2.Server`/`ReadIdleTimeout`/`PingInterval` 装配=**无腿
  可配的错配方向**——origin 升 h2/h2c 前禁装（T-0269 票面明列）。
- 真病灶放大器=前端无退避重连+重复取数（贴码修复见 T-0269）；
  残余风险=Watch 无应用层心跳时静默>100s 可能被 CF 杀流——20s
  保真帧关掉敞口。
- 会话级连坐签名（全请求同时挂+状态码 200）出现即先判连接级死亡，
  不逐请求排查。
