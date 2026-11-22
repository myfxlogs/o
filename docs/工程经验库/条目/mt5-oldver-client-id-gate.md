---
schema_version: 2
id: mt5-oldver-client-id-gate
kind: pitfall
risk: high
status: verified
keywords: [mtapi, OLD_VERSION, ConnectEx, id字段, 路由双面, CheckConnect毒化, AccountSummary活性探, 滚动半滚, cluster成员不齐, grpc2, grpc3, 契约反转, vendor断供]
scope: [internal/mt5, mtapi.io Connect/ConnectEx/CheckConnect, 会话生命周期, 共享会话复挂]
summary: mtapi 版本闸滚动部署半滚——ConnectEx 体字段自供 id 被路由未更新成员面全拒 OLD_VERSION，省略则网关自配 token 全通；CheckConnect 内部重登同撞闸对活 token 误拒——活性探改 AccountSummary 实数探
first_seen: 2026-11-20
last_verified: 2026-11-22
skills: [mt5-gateway-admin, mt5-account]
evidence_refs:
  - type: doc
    ref: ops_event#33036（三面初判行）+续行（变量隔离终判）
  - type: doc
    ref: internal/mt5/client.go:89（ConnectEx 去 Id 体字段——修后形）
  - type: doc
    ref: internal/mt5/recon.go:32（CheckAlive=AccountSummary 实数探——修后形）
  - type: commit
    ref: fix/mt5-oldver-dodge（绕行件——本坑根修）
---

## 事实/症状

MT5 面断供 55h+（行情末鲜停 2026-11-20 05:35Z）：生产 `ConnectEx`
全拒 `OLD_VERSION (:3001157)`，grpc2/grpc3 同病。vendor 称「已修复」
但本面三面对拍证伪：Version 串零变（`2026.10.14-10.44`）、
reflection 零漂移、生产实拍仍拒。

**关键不对称**：同源位同网关，裸 grpcurl ConnectEx 100% 成功，
生产客户端 100% 拒——唯一差异=生产请求**体字段 `id` 必填**
（`Id: &sid` 自供会话键）。变量隔离实证：

| 请求形 | 结果 |
|---|---|
| header `id=o-*` + 体无 id | ✅ token 回（复挂同会话） |
| header `id=o-*` + 体 `id=o-*` | ❌ `OLD_VERSION` 100% 复现 |

即**客户端自供 id 的请求被路由至未更新成员面**，网关自配 token
的新径已修复。CheckConnect 同毒：其内部走 `MT5API.Connect()`
重登撞闸——活 token 报 `OLD_VERSION (:0)` 而同 token
AccountSummary 返真账（$3247.31/hedging/real 实拍）。

grpc3 呈 OLD_VERSION↔`Server closed socket(send login)` 交替=
成员版本不齐半滚态（已修成员死更深段=11-09 出口拒再证）。

## 根因或策略解释

vendor 滚动部署按集群成员分批：API 层统一接请求，按 `id` 字段
路由到不同成员面——**自供 id 的请求绑定旧成员**（旧仿真端版本
被券商闸），新会话请求派往已更新成员。这是 vendor 侧 breaking
change 半滚态，非本方可控。

本方结构教训：**会话活性全架在 CheckConnect 单点上**——它内部
实现是重登（`GetClient→MT5API.Connect()`），版本闸一来活性探
与登录径同毒。「探针与被探对象共享失效模式」=单点同源病。

## 处置

- **Connect/ConnectEx 体字段 `id` 永禁填**（client.go 两处调用点
  已修）——网关自配 token 即会话凭，服务端同户同服去重，复挂
  天然零新 login；header `id` 保留无害。
- **活性探=AccountSummary 实数探**（recon.go CheckAlive 已换）——
  真数据路径证明真活，与登录径失效模式解耦。
- **版本核验勿信 Version RPC 串**——修复前后同报
  `2026.10.14-10.44`；核验=功能试（ConnectEx 实测）+reflection
  能力 diff。
- **vendor 回执径**：raw error 包已交业主转 mtapi.io（带 id 拒/
  不带 id 通/CheckConnect 内部误拒/AccountSummary 真账四联证）。

## 证据

- check: `grpcurl ConnectEx{user,pass,server}`（无 id）dev+SG 双源
  各 5/3 连发全成，复挂同 token `o-1794738961000294768`
- check: `grpcurl ConnectEx{...,id:o-*}` 100% 复现 `OLD_VERSION`
- check: `CheckConnect{o-1794738961000294768}`→`OLD_VERSION (:0)`
  vs `AccountSummary{同token}`→Balance 3247.31 real/hedging
- check: 生产 o-core 日志 55h 一致 `ConnectEx err: OLD_VERSION`
- ref: 券商日志——账户末次成功鉴权 11-16 18:34 自 91.98.38.182
  （Hetzner=mtapi 后端出口）；终端 build 6230 可登=闸认新版

## 反证条件

- falsifies_when: vendor 完成全量部署后，带体 id 的 ConnectEx
  恢复成功且 CheckConnect 对活 token 恢复 OK——此时「禁填 id」
  约束可复审是否仍必要（自配 token 本就更省心，建议保留）。

## 适用边界

- 适用：mtapi.io Connect/ConnectEx/CheckConnect 及一切经其会话的
  活性判定面（collector 复挂/sharedAttach/OpenSession）。
- 不适用：MT4 面（无 Version RPC，面形态不同——本案未取证）；
  ConnectProxy 径（未实测，逃生门候选在案）。
- 时效注：vendor 滚动未完——grpc3 出口对 Exness 有 11-09 拒连史，
  即便版本闸全修，grpc3 仍非可用径，生产锚 grpc2。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: mt5_tick_staleness
- class: 断供探针（既有探测面覆盖——venue dead 阈 2m0s 升件在案）
- query: `SELECT now()-max(recv_ts) FROM md_ticks WHERE venue='mt5'`
- threshold: >2m 升件（既有闸）
- 豁免由：探测面既有件覆盖（#33015/#33016 同簇升件实证触发）；
  本条目增量=根因定位与绕行修法，非新探测需求。
