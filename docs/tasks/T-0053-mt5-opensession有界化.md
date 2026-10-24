# T-0053 MT5 OpenSession 有界化+失败可见化

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` @ 2026-10-26（复审席 §11 部署实录移交件：SG 首建 MT5 会话静默悬挂） |
| 领取 | （空=未领） |
| 优先级 | 中 |
| 涉及面 | `internal/mt5/client.go`（Dial/Connect）、`internal/mt5/collector.go`（OpenSession）、`cmd/core/main.go`（装配调用点） |
| 范围外 | MT5 策略启用、OnOrderProfit 语义、网关侧问题排查 |

## 背景（复审席亲证）

SG 生产：汇市巡 roster Register+Beat 在岗，但 `md_account_events` 无 order_profit 订阅回执行——勘得 SG→mt5grpc3.mtapi.io:443 出口通、零会话终止零报错、启动日志止 4 行。**判定=OpenSession 静默悬挂**：`grpc.NewClient` 懒拨号+`Connect` RPC 吃调用方 ctx 无死线——首个 RPC 无限期阻塞于连接建立。

## 目标

会话建立全程有界+阶段可见——悬挂态在日志/告警面必须显形，禁静默占岗。

## 验收标准

- `Dial`+`Connect` 有死线（dial/RPC 各 ≤30s 或继承调用方更短 budget——宜 ctx.WithTimeout 包在 OpenSession 内，调用方免操心）
- OpenSession 分阶段 log：dial→connect→authed 每步起讫各一行；失败路径 log 含阶段名+err
- 悬挂可观察：超死线→WARN 日志+（有 alerter 则告警）+roster 态可查非 active
- 测试：假网关址（不可路由 IP 或本地黑洞口）注入→断言 OpenSession 在死线内返错而非悬挂
- `汇市巡` 重连语义不伤：超时返错后重试环如既有（若无重试环，本票内补退避重试——悬挂修复不能变成「一次败永久离线」）

## 验收证据位

测试锚+SG 部署后 `md_account_events` order_profit 订阅回执行出现（或显式 WARN 日志替代静默）。
