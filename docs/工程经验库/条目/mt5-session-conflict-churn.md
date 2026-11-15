---
id: mt5-session-conflict-churn
title: MT5 会话冲突掐——「流死≠会话死」复挂径与冷却自白退避
kind: pit
date: 2026-11-16
source: 升件#18354（venue 读败连败 mt5 10拍/41s，socket opened→send login→
  Server closed the socket）+ 本地 mt5connprobe 全谱实证（Exness/XM 双 venue）
---

# MT5 会话冲突掐（两 venue 同病实证）

**坑形**：OnQuote/推流断→立即 `ConnectEx`（每拍新 `o-<纳秒>` 会话 id）
→强开**新 login**→撞上**未死透的前会话**→券商在 login 段掐 socket：
`socket opened → send login → Server closed the socket`；mtapi 叠加
节流自白：`refusing logins. Connects to it are paused for N more
milliseconds`（Exness 词）/ `refused a login on every cluster member
just Nms ago. Please wait N milliseconds`（XM 词）。连败爆发数十秒，
尸寒（前会话过期）后自愈——**间歇性，非凭据/地址/网关坏**。

**判别**：错串含 `closed the socket`/`refusing logins`/`refused a
login`/`paused for`——会话冲突指纹。与 venue 无关（Exness·XM 同签名），
与接入 IP 无关（同池 4 址轮着掐），与凭据无关（同凭据闲拍全通）。

**修法**（实证序）：
1. **CheckConnect 复挂先行**——存上会话 (网关址, token)，重连先
   `CheckConnect(id=token)`：活=后端会话仍在，零新 login 直接重订流。
   实证：杀 gRPC conn 后 2s 新 conn 复挂 `check=OK`。
2. **冷却自白照睡**——错串内嵌 `paused for N ms`/`wait N ms` 直接解析
   睡 N+1s；socket-closed 无内嵌值睡 ≥12s（实测窗 5~8.3s 上界+余量）。
3. **会话键自持**——ConnectEx 体字段 `Id` 也填（不只 metadata id），
   复用旧 token 作键：grpc2/grpc3 皆回显自供键，复挂锚确定。
4. **流死不登出**——Disconnect 只在关停径；流死留会话给下轮复挂。

**错药实录**（全部证伪）：
- ConnectProxy/换出口 IP——拒因是账户会话级非 IP 级
- Bearer/付费 token——grpc3 免费道 Bearer 被忽略，拒因不在道
- 换网关（grpc2↔grpc3）——后端会话键不同但券商拒的是同账户
- 换券商（Exness→XM）——同死法同比例

**并发腿账**：O 有两条会话腿共用同账户 3001157——汇市巡（app_secrets
`mt5.*`）+ OCI 桥腿（app_config `bridge.mt5.*` 明文凭据）。两腿各自
churn+互撞=连败温床；复挂+冷却只能减 churn，跨腿撞是结构余账。

**推广**：凡「断线即新 login」的 venue 客户端（MT4/MT5/一切持会话协议），
重连第一问永远是「旧会话死没死」不是「新 login 能不能过」——
流死与会话死是两层，混为一层必产 churn。
