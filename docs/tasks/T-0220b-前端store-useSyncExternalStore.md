# T-0220b 前端 store.ts（useSyncExternalStore）+stream.ts v2 解析——useSnap 退役

| 字段 | 值 |
|---|---|
| 状态 | 施工中 |
| 立项 | `[角色:决策]` @ 2026-10-28（设计档 `docs/handoff/T-0173-最优解设计修订.md` §一.3/§二/§五；票面文件由施工席依决策席 dispatch+设计档原样物化——b46be9d 只落设计档未落票文件，来源留痕交验收核） |
| 领取 | glm-t0220 @ 2026-10-28 13:01Z |
| 优先级 | 高 |
| 涉及面 | `web/src/react/store.ts`（新件）、`web/src/react/stream.ts`、`web/src/react/hooks.ts`、消费组件（hostmetrics/flows/HedgeFlow/Overview/System/Review）、web/src/gen regen |

## 背景

T-0220a 后端推载荷后，前端仍逐卡 refetch 就浪费了。设计令：收帧解析 `data` → 写入全局 store → `useSyncExternalStore` 分片订阅精确重渲（每卡只订自己切片，只重渲该卡）；`useSnap` 的 useEffect+useState 样板退役。

## 目标

推帧→store 分片→组件自动重渲的一跳链路；三分片各有消费组件零死码；useSnap 全退役。

## 验收标准

- [ ] `store.ts`：全局分片 store（subscribe/getSnapshot/set/hydrate）+`useSlice`（useSyncExternalStore）——分片快照对象身份稳定（React 同时渲染一致性契约：set 才换身份）
- [ ] `stream.ts` v2：带载荷帧按 kind 解析（Struct→toJson→fromJson 类型化，零手拼）写入对应分片；无载荷帧保持 topic 信号语义；onLive 断流显形不变
- [ ] 三分片消费落位：`hostMetrics`→主机资源卡、`hedgeFlow`→统一流水默认视图（全部/页1）、`closedPositions`→历史订单首页（载荷 5 行对齐——卡翻页粒度 10→5 同票调整，偏离已注）
- [ ] 无载荷聚合面（overview/health/reports/equity/cashflows）保持信号重取语义迁移（useSnap 删除，消费点改 onData 信号径/分片径，零行为退化：错误显形、key 参数指纹保留）
- [ ] `tsc --noEmit`+`vite build` 绿；触碰文件 ≤300 行

## 范围外

后端/proto（T-0220a）；视觉层级（T-0220c）；玻璃材质（T-0220d）。

## 参考

- 设计档 §一.3/§二；T-0220a 载荷契约（ListHedgeFlowResponse/ListClosedPositionsResponse/GetHostMetricsResponse）
- 现状：`stream.ts`（onData/onLive 双订阅）、`hooks.ts`（useSnap）、消费点 grep useSnap=System×2/Overview×2/flows×1/hostmetrics×1/Review×3

## 结果回填

（施工写）
