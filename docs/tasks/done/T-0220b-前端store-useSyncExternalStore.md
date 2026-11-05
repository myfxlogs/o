# T-0220b 前端 store.ts（useSyncExternalStore）+stream.ts v2 解析——useSnap 退役

| 字段 | 值 |
|---|---|
| 状态 | 完成（判过并码在案，11-01清账翻面） |
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

- 实现摘要：`store.ts`（新件）——分片 subscribe/useSlice(useSyncExternalStore)/hydrate/set 三层；快照身份纪律（set 才换对象）+hydrate↔推帧竞态裁决=推帧优先（迟到回包弃陈）。`stream.ts` v2 ingest——gen 将 Struct 映 `JsonObject`，fromJson 类型化零手拼。`hooks.ts` useSnap→useLive（语义就义：推帧面归 store 分片，未载荷化聚合面必须保留信号重取径）。
- 消费迁移七点：hostmetrics=全分片；HedgeFlow 默认视图（全部/页1）分片+筛选/翻页信号重取；flows 历史订单首页分片+翻页 unary；Overview/System/Review useLive 改名迁移（行为不变）。
- 测试证据：`tsc --noEmit` 绿+`vite build` 绿；四页真数据目检（本地 core :50051+headless 截图）——系统页主机资源卡=hostMetrics 分片路径实数渲染（SG CPU 6%/内存 7.5/31.1GB），复盘页历史订单空态诚实显形。
- 偏离说明：①历史订单卡翻页粒度 10→5/页（对齐载荷 5 行，票面验收标准预注）；②useSnap 退役=删钩+未载荷化面以 useLive 名义保留信号重取（纯删会让 7 消费点失去更新机制；设计 §二「替代样板」针对推帧面——样板消除实现在三分片消费点零 useEffect/useState）。
- 坑处置：无坑。

---

## 验收注（`[角色:决策]` @ 2026-10-28，接管验收：原验收席 claude 卡住，业主令转本席；收口合并 `050b180`，实现件随 `26bf968` 在 master）

**裁：验收通过，票翻完成。** 票面物化来源追认（b46be9d 只落设计档未落票文件，施工依设计档 §一~§五 原样物化=合规）；回填四件（实现摘要/测试证据/偏离/坑处置）详实；偏离在案均认可；测试证据与生产目检（业主实见新前端）互证。

---

## 审计判决

审计席：headless @ 2026-10-31

逐锚对账（收口面=实现件 `26bf9686`/施工支头 `7879e949`，本票九件网摘 `7e27d777..26bf9686` 全量 diff）：

1. `store.ts` 全局分片 store——**过**。85 行新件实存；subscribe/useSlice/hydrate/set* 四面在件，`useSlice` 走 React 原生 `useSyncExternalStore`（三分片类型重载）。快照身份纪律实锤：`freshView()` 单对象、唯 `setView` 换身份（订阅方两次渲染间 getSnapshot 恒同对象）；hydrate↔推帧竞态=`pushed` 集合推帧优先（迟到回包弃陈）。锚列四件 subscribe/getSnapshot/set/hydrate 按能力面读——getSnapshot 内联于 useSlice 第二参、set 拆类型化 `setHedgeFlow/setClosedPositions/setHostMetrics`，实质满足。
2. `stream.ts` v2——**过**。`ingest` 按 kind 三分支 `fromJson(XxxSchema, ev.data)` 类型化解析（gen `data?: JsonObject`——Struct→JsonObject 映），零手拼；`if (!ev.data) return` 后 `subs.forEach` 无条件发 topic 信号——无载荷帧信号语义保持；`onLive`/`statusSubs`/`setLive`+`fn(live)` 回放原样未触，断流显形不变。
3. 三分片消费落位——**过**。`hostMetrics`→HostMetricsCard（useSlice+挂载 hydrate）；`hedgeFlow`→HedgeFlow 默认视图 `home=(src===""&&page===1)` 分片、筛选/翻页 unary+信号重取；`closedPositions`→ClosedPositionsCard 首页（`page===1`）分片、翻页 unary；`PAGE` 10→5 与后端载荷 `watch_payload.go` ev_position `Limit=5` 对拍一致（旁证：ev_cashflow `Limit=10`↔HedgeFlow `PAGE=10` 同律）。
4. 聚合面迁移零退化——**过**。`useSnap` 零码引用（仅存注释留名）；Overview×2/System×2/Review×3 消费点逐字改名 `useLive`，topics/fetcher/key 参数不动（`${period}:${page}`、`${kind}:${venue}:${page}` 指纹在）；`Live.err`→ErrorState 显形链未动，`useLive` 与旧 `useSnap` 实现逐句等价。
5. 机检绿+行数——**过**。CI `build-test` success @`7879e949`（`npm run build`=typecheck&&vite build 链）；合并冲突件仅 `docs/handoff/dialogue.md` 非码面；本票九件 85~262 行全 ≤300；origin/master 现行版分片码在册无回退（store.ts 97 行/hostmetrics 112 行持 useSlice+hydrate）。

非锚观察（报知悉，不构驳回）：`ev_cashflow`→topic `"trx"` 映射为 T-0156（e923d590）旧制——Overview/Review 以 `"ev_cashflow"` 为订阅键的聚合卡信号径自彼起即哑（消费方 `"trx"` 订阅实存于 TrxHedgeCard/TrxKpiStrip/TrxYieldRadar）；本票逐字同名迁移未改参数与匹配逻辑，零退化字面成立。建议另立单核修订阅键或映射面，不归本票。
