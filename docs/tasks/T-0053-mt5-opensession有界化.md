# T-0053 MT5 OpenSession 有界化+失败可见化

| 字段 | 值 |
|---|---|
| 状态 | 完成（判过并码在案，11-01清账翻面） |
| 立项 | `[角色:决策]` @ 2026-10-26（复审席 §11 部署实录移交件：SG 首建 MT5 会话静默悬挂） |
| 领取 | zcode-glm @ 2026-10-26（worktree /opt/phase-wt/t0053，支 t/0053-mt5-opensession） |
| 优先级 | 中 |
| 涉及面 | `internal/mt5/client.go`（Dial/Connect）、`internal/mt5/collector.go`（OpenSession）、`cmd/core/main.go`（装配调用点） |
| 范围外 | MT5 策略启用、OnOrderProfit 语义、网关侧问题排查 |

审计席：headless @ 2026-10-31

判决注记（逐锚独立对账，收口 commit 12874ac8 全量 diff 亲核+测试实跑）：
- 锚① Dial+Connect 有死线：**达**——Dial=grpc.NewClient 懒拨号即返（本身不可悬，真实建链发生在首 RPC）；Connect 调用方 ctx 无 Deadline→`connectBudget=30s` WithTimeout 兜底（client.go 新增段亲核），Server(ConnectEx)/Host 双轨同裹派生 ctx；OpenSession 内 WithTimeout(openSessionBudget=30s) 全包=锚钦定形（session.go）。潜态注：调用方 deadline 存在但 >30s 时不钳（存在性检非 min）——现全谱调用方（汇市巡 Run ctx 无死线/OpenSession 30s）皆 ≤30s 无现患。
- 锚② 分阶段 log：**达（义合）**——成径 dial起→connect起→authed 三行相位界；败径 WARN 携阶段名（dial/connect/creds）+err。严读「每步起讫各一行」未单列 dial讫/connect讫，相位起行兼作前相讫证，悬挂相位定位力（锚意）全具——录注不挡。
- 锚③ 悬挂可观察：**部分达**——WARN 字面 OpenSession 径实落；原事故径（汇市巡 session）超死线→`connect_failed` 计门 Inc+「会话终止（Xs 后重连）」日志——可见非静默但无 WARN 字面。alerter 子句空置：mt5 Collector/OpenSession 无 alerter 挂载点（struct 七域亲核；平台 alerter 在案仅挂 pgguard/ledger/warden）。**roster 非 active 子句未实装**——roster 纯活性模型（Register 钉 active/Beat 只刷 last_beat，roster.go 全文无非 active 写径），施工如实注记非本票面。**锚设计疑义报管理席裁决**：修后悬挂态灭，重试中工人 active=真实活性，venue 失联显形钦定面=计门+日志（D-652-A）；若钦要花名册映会话健康需另立票建降级态机制——翻 non-active 反谎报活性违数据单源真。
- 锚④ 黑洞注入断言死线内返错：**达（实测）**——`TestConnectDeadlineBounded` 黑洞址 10.255.255.1:443 双支（300ms 兜底注入/200ms 调用方 budget 继承）断言限时返错；本席实跑 `go test ./internal/mt5/ -run TestConnectDeadlineBounded -count=1` PASS。锚名「OpenSession」字面断在 Connect——Connect=懒拨号首 RPC=悬挂实点，OpenSession 为 readCreds(DB+master key)→Dial→Connect 薄壳，断 Connect=断悬挂机制于实点，义合录注。
- 锚⑤ 重连语义不伤：**达**——Run 退避环未触（reconMin 2s×2→reconMax 60s），session 返错→stream_dead 计门→日志→退避重试至 ctx 终，有界返错正喂既有重连非一次败永久离线。
- 宣称件三证齐：session.go 存 43 行/collector 294 行≤300 闸/connectBudget 落码/rejects 键集 connect_failed 在册/merge 623c18c4 在 master。SG 部署后 md_account_events order_profit 回执=票面明记归部署验证面，非本 commit 审计域。

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

---

## 结果（施工回填 2026-10-26）

### 码面
- `client.go Connect` 自带死线兜底（`connectBudget=30s` 包变量，测试可短缩注入）：调用方 ctx 无 Deadline 时 `WithTimeout` 兜底；已带更短 budget 继承不覆写。根修点=懒拨号首 RPC（Connect/ConnectEx）——一处修覆盖 OpenSession/Collector.session/探针全部调用径。
- `session.go` 新件（collector.go 触 300 行棘轮故拆）：`OpenSession` 全程死线 `openSessionBudget=30s`（ctx.WithTimeout 包内，调用方免操心）+分阶段 log——dial 起/败、connect 起/败（含 addr+「超死线/登录拒非悬挂」注记）、authed 讫；失败返错带阶段名（`阶段=dial:` 等）。
- 悬挂可观察：超死线→WARN 日志+返错；Collector.session 连接失败走既有 `connect_failed` 计门。**重连语义不伤**：session 外层 Run 循环既有退避重试（reconMin×2→reconMax）原样保留——超时返错即进退避重连，不会一次败永久离线。
- roster 态注记：汇市巡 Register+Beat 在会话建前已上岗（既有语义），悬挂修复后 connect 败在日志/计门显形——roster 态本身不因会话败翻非 active（BeatLoop 常跳），如实注记非本票改动面。

### 测试锚（绿）
`TestConnectDeadlineBounded`（黑洞址 10.255.255.1:443 注入）：① 无 Deadline ctx+300ms 兜底死线→返错且 <5s（实证真悬挂至死线才解——修前此处永久阻塞）；② 调用方 200ms 短 budget→继承返错 <2s。

### 机检
build/vet/gofmt/test 全量 / -race（mt5）/ import-direction / migration-contract / lines（collector 279、session 43）/ env-reads——全绿。无新 migration。

### 验收证据位
测试锚已落；SG 部署后 `md_account_events` order_profit 订阅回执行出现（或显式 WARN 替代静默）——归决策席部署验证。

### 状态：待验
