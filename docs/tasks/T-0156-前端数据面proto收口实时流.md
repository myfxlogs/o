# T-0156 前端数据面 buf/proto 收口+实时流化（业主 10-26 双令：禁轮询、废手拼 JSON）

| 字段 | 值 |
|---|---|
| 状态 | ✅ 验收毕投产（决策席 2026-10-26） |
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
