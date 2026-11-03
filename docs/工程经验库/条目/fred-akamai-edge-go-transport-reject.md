---
schema_version: 2
id: fred-akamai-edge-go-transport-reject
kind: pitfall
risk: high
status: verified
keywords: [FRED, Akamai, HTTP2, stream INTERNAL_ERROR, Go JA3, curl http1.1, api_key 日志泄露, ASN 歧视, 备用径]
scope: [research, ctx, OCI, http-transport, 公开面采集]
summary: 真凶=自造 UA（o-ctx/o-research）被 Akamai 边池拉黑——h1 tarpit 挂死+h2 stream 踢流；四 UA 隔离实证 curl/Go 默认 UA 秒通。「h2 间歇/边池异态」是 UA 病表象；err 携 api_key URL 落日志必脱
first_seen: 2026-11-03
last_seen: 2026-11-03
evidence_refs:
  - type: ticket
    ref: T-0311
  - type: commit
    ref: t/0311-regime-analyst
---

## 定案（四 UA 隔离实验 13:44）

curl 默认 UA / Go 默认 UA（Go-http-client/2.0）= 200 秒通；
`o-ctx/1.0` / `Mozilla/5.0 ... o-research/1.0` = h1 tarpit 挂死 20s+h2
stream INTERNAL_ERROR——**拦截对 UA 不对客户端栈**。修法=撤自造 UA
用栈默认。fred 四序列（DGS10/DGS2/UNRATE/CPIAUCSL）落行实证。

## 误判链实录（防再犯教训件）

初判「Go JA3 指纹」「h2 协议坏」「边节点异态」「单 IP 无轮换空间」——
全是 UA 病的表象：探针从没设自定义 UA（默认 UA 侥幸全中），把锅甩给
了传输栈。**隔离变量不够狠=鬼打墙**：客户端栈/协议/节点全排完才轮到
UA——下次变量矩阵把 UA/请求头排进第一轮。

## 实击链（2026-11-03 T-0311 部署窗口，OCI 出网）

- **h2 协议级坏窗口**：Go h2 默认 client 间歇 `stream INTERNAL_ERROR
  received from peer`；**curl --http2 同窗口同样 INTERNAL_ERROR**——
  不是 Go JA3 指纹恒定拦，是 Akamai 边对 h2 协议当下即坏。
- **单 IP 边池**：api.stlouisfed.org DNS 只返一 A 记录
  （23.15.108.195 akamaiedge）——多 A 轮换拨号对此域无效。
- **手搓 Transport 静默降 h1**：`&http.Transport{}` 不带
  ForceAttemptHTTP2——DefaultTransport.Clone() 或显式置位才保 h2；
  Go h1 在该边池挂死（30s Client 超时实击），**curl --http1.1 同端点
  400/502/400 抖动但主体通**——h1 是当前唯一可靠径。
- **api_key 落日志**：err 携完整 URL（api_key 明文）被 log 打出，
  半键已落 journal——`*url.Error` 必须剥壳只留内层。

## 修法（ctx httpGet 现形）

传输类哨兵 `errTransport` 分类：h2 三试（status/建请败不重试）→ 尽后
`curl --http1.1` 备用径；30s Client 总超时；err 脱 URL 壳。
6h 拍 fail-soft——边劣化窗口缺席不炸，边恢复自落。

## 伴生谱系（机房 ASN 歧视同族）

mql5 整站 403 死源（已弃换 ffcal）；ffcal 对 UA `o-ctx/1.0` 429、curl
默认 UA 通；FRED 边传输踢流。**云机房 ASN 出网采公开面先天打折——
备用径+分类重试是指定药**。

## 证据

- check: Go h2/curl h2/curl h1 三径同端点分窗口实测；DNS getent 单 A
  记录；journal `[ctx] fred 拉取败` 灯行序列
- result: h2 双栈同 INTERNAL_ERROR；h1 curl 400/502 主体通；Go 探针
  进程间或中（窗口依赖实锤）
- ref: T-0311 返修⑤ commit 链（AttachFredFn/日志灯/Clone 保 h2/RR
  拨号弃/curl h1 备径）

## 反证条件

- check: 连续三日 fred 行 ts 递增零败
- falsifies_when: 用默认 UA 仍被拦=UA 假说证伪（当判：默认 UA 全通已证 UA 是根因）
