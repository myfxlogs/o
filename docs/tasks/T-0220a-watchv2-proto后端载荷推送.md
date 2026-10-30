# T-0220a Watch v2——proto data 载荷字段+后端按 kind 查表推数据

| 字段 | 值 |
|---|---|
| 状态 | 审计通过 |
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

- 实现摘要：`core.proto` WatchEvent 增 `google.protobuf.Struct data = 3`（buf 双模板 regen 过）；`cmd/core/watch_payload.go`（新件 50 行）——kind 路由复用既有读面方法零重写查询（ev_cashflow→ListHedgeFlow limit 10 / ev_position→ListClosedPositions limit 5 / ev_hostmetrics→全量），protojson EmitUnpopulated→structpb 类型化直转；`service.go` Watch 循环挂载荷（查/转败=帧照发+log 显形，watch_open/ev_account/ev_strategy 纯信号不变）。
- 测试证据：`watch_payload_test.go` 三锚全绿（ev_position 载荷=最新 5 行闭仓倒序+total 反读同值；signal kinds=nil 纯信号；ev_hostmetrics 零行=显式空集）；`TestConnectEndToEndAndWatchLatency` 增载荷断言（ev_cashflow 帧必带 txs 数组）PASS（O_TEST_PG_DSN 5434 实测，`go test ./cmd/core/ -run TestWatchPayload`+过线桩锚均 ok）。
- 偏离说明：①`EmitUnpopulated:true`——默认 protojson 省零值字段会让空响应载荷成 `{}`（前端零键可解），显式空集契约更稳；②fin_position 测试行需 execution 父行+notional_usd（迁移形演进，RETURNING id 回填不假设序列值）。
- 坑处置：无坑。

---

## 验收注（`[角色:决策]` @ 2026-10-28，接管验收：原验收席 claude 卡住，业主令转本席；收口合并 `050b180`，实现件随 `26bf968` 在 master）

**裁：验收通过，票翻完成。** 票面物化来源追认（b46be9d 只落设计档未落票文件，施工依设计档 §一~§五 原样物化=合规）；回填四件（实现摘要/测试证据/偏离/坑处置）详实；偏离在案均认可；测试证据与生产目检（业主实见新前端）互证。

---

## 审计注记

审计席：headless @ 2026-10-30

**判：审计通过。** 收口实现件 `ba8f81d`+`6b958b1`（随合并 `26bf968` 落 master）全量 diff 对票面五锚逐条实核，隔离 worktree（基 origin/master @5073982）复跑实证：

1. **proto 载荷字段+双模板 regen**——PASS。`core.proto` `import "google/protobuf/struct.proto"` 在册、`WatchEvent` 增 `google.protobuf.Struct data = 3` 在册（字段号未挤占 kind=1/ts=2）；审计 worktree 实跑 `buf generate` 三插件 exit 0——`core.pb.go:641` `Data *structpb.Struct`、`core_pb.ts:348` `data?: JsonObject` 双双再产（Go+web 双模板实证非宣称）。
2. **kind 路由查表推载荷**——PASS。`watch_payload.go` switch 三分支对位：`ev_cashflow`→`ListHedgeFlow{Limit:10}`（源径 trx_faces.go:91→`ListHedgeFlowPage(limit)`）、`ev_position`→`ListClosedPositions{Limit:5}`（host_face.go:45-60 SQL `ORDER BY closed_at DESC LIMIT $1`——最新 5 行语义实）、`ev_hostmetrics`→`GetHostMetrics{}` 全量；全调既有读面方法零重写查询；序列化径 protojson EmitUnpopulated→`structpb.UnmarshalJSON` 类型化直转（禁手拼 JSON 律守）。
3. **watch_open/信号面不变**——PASS。`service.go` `watch_open` 帧于 LISTEN 注册后循环前直发仅 `{kind,ts}` 零载荷（diff 未触该段）；`watchPayload` default→`(nil,nil)`，`ev_account`/`ev_strategy`/`watch_open` 纯信号帧——`TestWatchPayloadSignalKinds` 三 kind nil 断言实跑绿。
4. **查/转败帧照发+log 显形**——PASS。`service.go` `perr != nil → log.Printf("watch 载荷 %s 查败（帧照发）: %v")` 后 `stream.Send(ev)` 照发；查败与 Marshal/UnmarshalJSON 败同归 perr 分支——信号语义保留、不炸流、不静默吞。
5. **build/vet/test 绿+行数**——PASS。审计 worktree `go build`/`go vet ./cmd/core/` 双绿、gofmt 零漂移；`O_TEST_PG_DSN`@5434 实测 `TestWatchPayload`×3 + `TestConnectEndToEndAndWatchLatency`（含 `6b958b1` 增 ev_cashflow 帧必带 `txs` 载荷断言）4/4 PASS。行数：service.go 273、watch_payload.go 50、watch_payload_test.go 98 全 ≤300；载荷逻辑独立成件=锚括注实旨全达。

附记：触碰件 trx_test.go 332 行>300——该件属 `*_test.go`，check-lines.sh `exempt()` 明豁免，且本票前已 327 行存量（+5 行载荷断言）；按门禁契约口径计过，留注备查。
