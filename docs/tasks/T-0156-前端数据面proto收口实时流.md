# T-0156 前端数据面 buf/proto 收口+实时流化（业主 10-26 双令：禁轮询、废手拼 JSON）

| 字段 | 值 |
|---|---|
| 状态 | 完成（判过并码在案，11-01清账翻面） |
| 立项 | `[角色:决策]` @ 2026-10-26（业主令：①禁轮询/定时任务要实时流 ②手拼 JSON 易错走 buf proto） |
| 领取 | glm-t0156 @ 2026-10-26 |
| 优先级 | 高 |
| 涉及面 | `proto/dashboardv1/core.proto`（扩面）、`cmd/core/service.go`/`trx_faces.go`/`trx_yield_face.go`（REST 手拼面撤）、`web/src/react/api.ts`/`stream.ts`、`web/index.html`+`web/src/main.ts`（30s 轮询撤）、buf 双模板 |

## 背景（评估实证，2026-10-26 前端评估）

手拼 JSON 当场炸出生产现行 bug：后端 map 键 `energy_apr_7d`
（trx_yield_face.go:48）数字后缀，前端 `camel()` 正则 `/_([a-z])/g`
不含数字——**收益率卡 7d/30d 四格生产恒显「—」**（本地渲染实锤；
「更新于 Invalid Date」同因族：string ts 喂 proto Timestamp 读取件）。
根因=同数据双轨两份序列化（`/api/*` 手拼 REST + `CoreService` proto
Connect 挂载同 mux）人肉对拍键名。后端 Watch 流（LISTEN/NOTIFY ev_*）
早已在案，前端却走 SSE shim+**两处 30s 轮询残留**（main.ts:106、
stream.ts:11）违业主事件驱动令。

## 目标（做完后世界）

- proto 单源：TRX 四读面+监控面入 `dashboardv1` proto，buf 双模板
  regen Go+TS；`api.ts` REST shim 撤，`@connectrpc/connect-web` 直连
  `CoreService`（组件零感设计兑现——T-0113 铺的正是这条路）
- 实时流：数据更新唯一驱动=`CoreService.Watch` 流帧（ev_* 脊），
  **30s 轮询两处+SSE `/api/watch`+REST `/api/overview` 全撤**
  （时钟 1s interval 非数据面，保留）
- 监控提醒卡：后端 watch_rule 面未落位——**撤卡**（假功能下架，
  UI 不得承诺系统能力外的事；后端落位后同 proto 恢复）

## 施工步

0. **热修先行**（不等整单）：`api.ts` camel 正则 `_([a-z0-9])`——四格
   当天复活；`ts()` 兼容 RFC3339 string。可独立 commit 先投产
1. proto 扩面+buf regen（Go+TS 同 commit）
2. 后端 trx_faces/trx_yield_face 改实现 proto service（挂 connect mux），
   撚 REST handler
3. 前端 connect-web 接管+Watch 流驱动+撤轮询/SSE/监控卡
4. `/api/overview` 撤（总览页 vanilla main.ts 同轮切 connect-web 或并
   React——并线另议，本单只撤轮询保持 vanilla 面活）

## 验收标准

- [ ] 收益率卡六格真值显形（本地+SG 双端；_7d/_30d 格复活）
- [ ] 页面数据更新仅由事件驱动：backend NOTIFY→前端 <1s 显形（桩锚）
- [ ] `grep -rn 'setInterval' web/src/` 仅剩时钟两处；`/api/overview`、
      `/api/watch`、`/api/trx/*` REST 面 全撤（404 化）
- [ ] 手拼 JSON 面零残留（trx_faces.go/trx_yield_face.go 撤或改 proto 实现）
- [ ] 监控提醒卡不渲染；buf gen TS 类型零本地 interface 等价件残留
- [ ] build✓ typecheck✓ test✓（Go+web 双侧）

## 范围外

- 总览 vanilla 页整体迁 React（另议）
- watch_rule 后端面落位（另票）
- 美观/响应式缺陷簇（T-0157）

## 参考

- `cmd/core/service.go:149`（Watch 流——事件脊现成）、`:200`（REST 双轨）
- `web/src/react/api.ts:14`（camel 正则病灶）、`web/src/react/stream.ts:11`（保底轮询）
- `cmd/core/trx_yield_face.go:48`（手拼 map 病灶）
- 评估截图：/tmp/o-render/（ov-dark/light/narrow、trx-light）

## 结果回填（glm-t0156 @ 2026-10-26）

- **步0 热修独立投产**（529c848）：`camel()` 正则 `_([a-z0-9])`——
  `_7d/_30d` 数字尾缀键复活；`ts()` string 支路——「更新于 Invalid Date」
  根治（步3 收口后 string 支路已随 proto 直连撤除，热修语义由 proto
  Timestamp 直读继承）。
- **proto 单源**（core.proto 扩面）：CoreService 新增七 RPC——
  GetTrxAccount/TrxAccountRefresh/ListTrxTxs/GetTrxYieldRadar/
  GetTrxYieldCurve/GetHedgeStructure/ListHedgeFlow；消息形状对齐消费面
  （折值后端算定，found=false=读件缺显形不假零）；buf 双模板 regen
  Go+TS 同 commit（101c919+本票）。
- **后端收口**：trx_faces.go/trx_yield_face.go 改 proto service 实现
  （trxWiring 同源注入 coreService）；computeHedgeCarry→typed
  hedgeCarry、legQtyStaked 撤；mountTrxAPI/writeJSON/cmc_history/
  cex_scan REST 面全拔；mountUI 砍成纯静态（/api/overview+/api/watch
  SSE 撤，404 化有桩测锚）。**偏离说明**：票面「四读面」实落六读面+
  刷新踢——GetTrxAccount/ListTrxTxs 为 vanilla 总览 TRX 段活口所需
  （原段消费的 radar/vote/cmc/cmc_positions 僵尸 face 自 T-0110 起即
  404，本票顺带修活），验收「/api/trx/* 全撤」的数学下界。
- **watch_open 就绪帧**（service.go Watch）：connect 服务端流首写前不
  发响应头——客户端 Watch()/for-await 悬挂至首事件（潜伏雷：该服务
  此前从未被前端真用过）。LISTEN 注册后发 watch_open 非数据帧=就绪
  信号+前端连接即回填。坑条目同 commit 入库。
- **前端 connect-web 接管**：@connectrpc/connect(-web)@2.2.0 直连
  CoreService（api.ts 同签名件——组件零感兑现 T-0113 铺路）；双页
  （vanilla index+React trx）全切；react/gen 本地 interface 等价件三件
  删（trx_cex_pb/position_pb/watch_rule_pb）+TrxWatchCard 监控卡下架
  （watch_rule 后端落位后同 proto 恢复）；stream.ts=Watch 流帧泵
  （ev_cashflow→trx 主题映射——TRX 现金行租金/领奖/隔夜息全在
  fin_cashflow），30s 保底轮询+SSE shim 撤；vanilla main.ts 30s 轮询+
  EventSource 撤（watchPump 3s 断流重连=传输维护非数据轮询，注释在
  案）；renderTrx 改 proto 读面活口（单面败不连坐）。
- **验收锚对账**：
  - 收益率卡六格真值显形：码面达成（energyApr7d/30d 数字尾缀键
    proto 直供）——本地浏览器实证未做（本地无 o-core 全栈运行面），
    **SG 双端显形=验收轮换后决策席实证项**。
  - 事件驱动 <1s：桩锚 TestConnectEndToEndAndWatchLatency（-count=3
    确定性过）——watch_open 就绪帧后拍行，ev_cashflow 帧达 <1s 实测
    ms 级；前端帧→重取接线码面直驱（流帧同步派发）。
  - setInterval 仅剩时钟两处：grep 实证（Shell.tsx:16+index.html:86
    各 1s 时钟）；/api/overview、/api/watch、/api/trx/* 404 化：桩测
    锚 TestRestFacesRetired。
  - 手拼 JSON 零残留：trx_faces.go/trx_yield_face.go=proto 实现；
    writeJSON/mountTrxAPI 拔除（grep 零命中）。
  - 监控提醒卡不渲染+本地 interface 零残留：trx.tsx 撤卡；
    react/gen/ 整目录删。
  - build✓ typecheck✓ test✓：go build/vet/gofmt 净+go test ./...
    全量绿（四 DSN 真跑，exit 0）+web typecheck/build 净。
- **机检**：build✓ vet✓ gofmt✓ test✓（全仓四 DSN）buf lint✓（新增
  面零告警，存量历史债另案）web build✓ typecheck✓。

审计席：headless @ 2026-10-30

## 判决注记（无头审计——逐锚对账收口链 529c848a→101c9195→e923d590→d8de5abb 全量 diff）

- **锚①收益率卡六格真值显形（_7d/_30d 复活）**：根修成立——proto
  `energy_apr_7d/30d` 类型化字段直供（core.proto:111-112），前端
  `radar.energyApr7d/30d` 直读（TrxYieldRadar.tsx:45/66/75），snake→camel
  正则盲区随双轨序列化撤除而根除；步0 热修 `_([a-z0-9])` 正则先行在案
  （KB 条目 snake-camel-regex-digit-suffix-blindspot 同 commit 入库）。
  本地 DB 侧只读探查 fin_trx_account/vote_apr/txs 三表本日续产=数据面有料。
  施工如实申报本地浏览器实证未做、SG 显形归验收轮换决策席实证项——
  该票当时已验收合入投产翻牌。码面机制可证+实证责任归属在案。**PASS**。
- **锚②事件驱动 <1s**：Watch 实现=LISTEN ev_position/cashflow/account/
  strategy 四通道→watch_open 就绪帧→WaitForNotification→Send 直转
  （service.go:151-178）；桩锚 TestConnectEndToEndAndWatchLatency 断言
  NOTIFY→帧达 <1s+首帧 watch_open+通道 kind（trx_test.go:225-269）；
  前端 main.ts watchPump→ov() 帧驱重取+stream.ts pump→subs 派发。
  **PASS**。
- **锚③setInterval 仅剩时钟两处+/api/* 404**：收口树 grep setInterval
  全仓=index.html:86+Shell.tsx:16 各 1s 时钟两处（web/src/ 内仅一处，
  锚文「两处」口径含 index.html——时钟豁免项票面已注）；/api/overview·
  /api/watch·/api/trx/* 零注册——mountCore 仅挂 Connect handler、mountUI
  纯静态 embed，TestRestFacesRetired 桩测六路径 404 断言在案；
  EventSource/fetch 数据径零命中。main.ts/stream.ts 3s setTimeout=断流
  重连传输维护（注释在案），非数据轮询。**PASS**。
- **锚④手拼 JSON 零残留**：trx_faces.go/trx_yield_face.go=CoreService
  proto 实现（typed response，found=false 读件缺显形）；writeJSON/
  mountTrxAPI 全仓零命中；cmd+web 双侧无手拼 JSON 写径。**PASS**。
- **锚⑤监控卡不渲染+interface 零残留**：TrxWatchCard.tsx 整件删（152 行）、
  trx.tsx 仅渲染三卡；web/src/react/gen/ 目录三件（trx_cex_pb/position_pb/
  watch_rule_pb）全删、目录不存；组件 import 全指 ../../gen/dashboardv1/
  core_pb（buf 产物 gitignored）。**PASS**。
- **锚⑥build/typecheck/test**：收口时代无 CI required-check（10-27 令前），
  绿证=验收合入 commit「构建实证绿」+回填全量绿申报；本席按钦定禁跑
  check_all（并发互踩共享测试库），改以静态对账：proto↔后端实现字段名/
  proto↔TS 消费面字段名逐点一致、七 RPC 注册齐全、依赖件
  @connectrpc/connect(-web)@2.2.0 在 package.json、桩测三件在树。
  **PASS**。
- **范围偏离裁**：票面「四读面」实落七 RPC——施工自报 GetTrxAccount/
  ListTrxTxs 为 vanilla 总览 TRX 段活口所需（原 REST face 自 T-0110 即
  404 僵尸），dialogue.md 偏离自报+决策验收合入追认在案，属合规偏离。
- **非锚注记**：TrxAccountRefresh 内 `_, _ = w.accts.Poll`/`floorPct, _ :=`
  静默吞错——读面刷新踢非钱径、注释声明静默败不阻断，票面无此锚，记档
  不阻通过。后票演进（T-0220 系 Watch v2 载荷直推等）不溯本票对账。
