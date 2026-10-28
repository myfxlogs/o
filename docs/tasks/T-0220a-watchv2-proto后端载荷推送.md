# T-0220a Watch v2——proto data 载荷字段+后端按 kind 查表推数据

| 字段 | 值 |
|---|---|
| 状态 | 施工中 |
| 立项 | `[角色:决策]` @ 2026-10-28（设计档 `docs/handoff/T-0173-最优解设计修订.md` §一/§五；票面文件由施工席依决策席 dispatch+设计档原样物化——b46be9d 只落设计档未落票文件，来源留痕交验收核） |
| 领取 | glm-t0220 @ 2026-10-28 13:01Z |
| 优先级 | 高 |
| 涉及面 | `proto/dashboardv1/core.proto`、`cmd/core/service.go`（Watch 循环）、`cmd/core/watch_payload.go`（新件）、gen 双模板 regen |

## 背景

现状 Watch 流是两跳：PG NOTIFY → 推 `{kind,ts}` 纯信号帧 → 前端收帧逐卡发 unary 重拉 → 渲染。每次更新多付一趟往返（~100-200ms），且前端逐卡各自 fetch+state。设计档业主令=最优解：**一跳**——NOTIFY 到达时后端直接查对应表最新行，序列化进载荷字段推入流。

## 目标

`WatchEvent` 带 `google.protobuf.Struct data` 载荷；后端 Watch 循环按 kind 路由查表推数据；前端（T-0220b 消费）收帧即渲染零重拉。

## 验收标准

- [ ] proto：`WatchEvent` 增 `google.protobuf.Struct data = 3`（import struct.proto），`buf generate` Go+web 双模板过
- [ ] 后端 Watch 循环按 kind 路由推载荷：`ev_cashflow`→ListHedgeFlow 最新 10 行、`ev_position`→ListClosedPositions 最新 5 行、`ev_hostmetrics`→GetHostMetrics 全量（复用既有读面方法，零重写查询）
- [ ] `watch_open` 初始帧语义不变（就绪信号，无载荷）；`ev_account`/`ev_strategy` 保持纯信号帧
- [ ] 载荷查询/序列化败=帧照发+log 显形（信号语义保留，不炸流、不静默吞）
- [ ] go build/vet/test 绿；触碰文件 ≤300 行（载荷逻辑独立成件，service.go 不超限）

## 范围外

前端消费面（store/stream/组件——T-0220b）；视觉面（T-0220c/d）；ev_account/ev_strategy 的载荷化（设计档未列，保持信号）。

## 参考

- 设计档：`docs/handoff/T-0173-最优解设计修订.md` §一
- 现状码面：`cmd/core/service.go:182`（Watch 循环）、`cmd/core/trx_faces.go:91`（ListHedgeFlow）、`cmd/core/host_face.go:13/45`（GetHostMetrics/ListClosedPositions）
- 序列化径：protojson.Marshal → structpb.Struct UnmarshalJSON（类型化消息直转，禁手拼 JSON 串）

## 结果回填

（施工写）
