---
name: mt5-gateway-admin
description: mtapi 网关连接/服务面——Connect token 语义/Service 族/reflection dump/权威源对拍纪律
domain: 工程
---

# mt5 网关服务面（实测 2026-10-17，证据=docs/能力账/mt5-rpc-probe-2026-10-17.json）

| RPC/面 | 功能 | 实测 | 关键坑 |
|---|---|---|---|
| Connection.Connect | 连接+登录（返回 token） | 三户 ok（256-500ms） | **token 语义（2026-10-24 修正）**：**header `id` 必带**——Connect 时 metadata `id=<临时 sid>`，登录后**所有**调用（unary+stream 同）metadata `id`=<token>，缺之全拒 InvalidArgument「Id header is required」；请求体 `Id` 字段同填不替（nox 期「header 径不读」判词失效——EXR 3001157 实拍反证）；多 fallback 端点 mt5grpc1/4/2.mtapi.io |
| CheckConnect | 连接活性 | 独立连接 id 径（probe skipped 如实） | 断线自愈=adapter sessionClient 重连包装 |
| ConnectEx/ConnectProxy | 连接变体（扩展参数/代理） | **ConnectEx 2026-10-24 实测 ok**（EXR 3001157 真户，server 名径「Exness-MT5Real」一次通） | ConnectEx{User,Password,Server} 与 Connect{Host,Port} 双径皆可用；ConnectProxy 未实测 |
| PingHost(Many) | 服务器端点探活 | PingHost ok；Many 版 err「host.Length != port.Length」 | Many 版入参=host/port 配对数组，参数形态坑在案 |
| GetClients/MemorySnapshot | 网关管理面 | GetClients err「Unable to cast…」依旧；**MemorySnapshot 2026-10-24 恶化为挂起超时**（不再快错——调用方须带 deadline） | 管理面不可用如实——且 MemorySnapshot 现为**悬挂型**缺陷（更危险） |
| Disconnect | 断会话 | probe 收尾在用 | — |
| Service.Ping/Version | 网关活性/版本 | ok——2026-10-24 实拍 Version=**2026.10.14-10.44** | 无 token 面 |
| Service.Health/GetDemo | — | Health=Unimplemented；GetDemo=side-effect（开 demo 户）标缺 | GetDemo 会真开 demo 账户——业主令禁自开户 |
| Service.Search | 公司→服务器/地址解析 | ok | demo 端点正径：`Search(company="Exness")`→Results[].Access 列表（**host:port 须拆分**——整串当 host=Connect disposed/timeout 实拍）→Connect 户主凭据；实拍 MT5Trial5=18.163.85.196:443 |
| reflection | 服务/方法/消息/枚举全 dump | 证据 JSON `.reflection` 节 | **补「名单 grep」缺口**的正道——新增网关版本能力 diff 用它 |
| 权威源 | /opt/grpc-proto/mt5/protos/mt5.proto | 2026-10-17 对拍：仓内副本 30 行 diff 全注释级（go_package 定制+ClusterDetails 注释块）——枚举/语义零漂移 | 仓内副本随权威源修（D-604 对拍纪律） |

**能力定谳三足**：官方清单数（reflection+权威源 proto）×实测证据（能力账 JSON）×skills 正典（本目录）——三者同源对拍。

<!-- 方法全名索引：Connect ConnectEx ConnectProxy CheckConnect Disconnect Ping PingHost PingHostMany Health MemorySnapshot Search GetClients GetDemo Version -->
## 装配面（重建件——skills 原只有 RPC 面）
- **网关配置键**：`NOX_MT5_GATEWAY`（缺省 `mt5grpc3.mtapi.io:443`）+`NOX_MT5_GATEWAY_FALLBACKS`（缺省 `mt5grpc1,mt5grpc4,mt5grpc2.mtapi.io:443`）——`internal/platform/config/config_mt_health.go`。
- **adapter 层**：`internal/data/adapter/mt5.go`——`NewMT5Adapter(venue,host,server,port,user,pass)` 每户一适配器；conn=google.golang.org/grpc。
- **会话生命周期**：`mt5_session.go reconnectSession`——复用 conn 重调 Connect 刷 token（token 即 Request.Id）；`OnReconnect` 钩=对账 kick 单点（D-004）；token 日志须 maskToken。
- **生成层**：`internal/data/adapter/gen/mtapi/mt5`——mt5.pb+grpc+mt5connect 三件（权威源 `/opt/grpc-proto/mt5/protos/mt5.proto` 对拍纪律在案）。
## 凭据装配链（ant gateway/mt5 实证移植——重建第一日件）
- **Credentials 字段**：`{MTType, Login(string→uint64), Password, Server, Host, MtapiToken}`；Endpoint.Addr=mtapi 网关址（非 broker 址——broker 址在 Host/Server 里）。
- **双轨寻址**（实证）：`Server` 非空→`ConnectEx{User,Password,Server,Id}`〔server 名=MT 终端所见名〕；否则 `Host`→`Connect{User,Password,Host,Port,Id}`——Host 为**逗号分隔多候选**逐个试〔broker_search 搜得的 Access 列表直接灌〕，`splitHostPort` 末 `:` 剥端口、缺省 **443**。
- **id/token 双层**（坑密集面，2026-10-24 修正）：Connect 前先以 metadata `id=<临时 sid>` 发起；Connect 返回的 token 回填**两处**——metadata header `id` **且** 请求体 `Id` 字段（**双位同填，缺一即拒**——EXR 3001157 实拍：仅体域无 header=「Id header is required」unary/stream 同病）。**另轨**：流/订阅类调用可能还须 `authorization: Bearer <MtapiToken>`——MtapiToken 空=免费公共端点轨省发（公共端点不校验 Bearer 实拍），付费/私有端点必须带。
- **dial 参数**（ant 实证集）：TLS1.2+、`MaxCallMsgSize 16MB`、keepalive `30s/20s/PermitWithoutStream`（**长流必须**，无则被中间件静默割）。
- **Disconnect 排水窗**：Disconnect RPC 后等 ~200ms 再 `cc.Close`——让在飞 Recv 观察到取消，防级联重连（ant QUOTE-RECONNECT-LOOP 实证）。
- **Ping**=`CheckConnect{Id}`；**登出纪律**：Disconnect 只应由持有方 actor 发起。
