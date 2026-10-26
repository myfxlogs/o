# T-0156 前端数据面 buf/proto 收口+实时流化（业主 10-26 双令：禁轮询、废手拼 JSON）

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` @ 2026-10-26（业主令：①禁轮询/定时任务要实时流 ②手拼 JSON 易错走 buf proto） |
| 领取 | （空=未领） |
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
