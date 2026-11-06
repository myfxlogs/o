# T-0378 hungertick 接线——饥饿感知器挂 lifecycle 周期拍

- 状态：待验（2026-11-06 外派 run 收口——自测全绿，移交决策席常径裁）
- 领取：业主手动外派 @ 2026-11-06（外部工人 run——提示词
  docs/prompts/T-0378-O-Dev施工提示词.md，DM 勿自动领；
  run=claude-t0378 @ 2026-11-06 领单开工）
- 优先级：P1——T-0376 自治环唯一未接活件
- 收口移交归：决策席（外派件收口回 DM 常径裁——86efad3 新规）

## 病灶

`internal/portfolio/hunger`（T-0376 落地：Detect 两触发形+
Respond 四步响应+DMSet 参数调整行）**零 caller=死件**——
cmd/ 无入口、dm_tick 无调、lifecycle 无挂。饥饿环机制在库
不跑。

## 范围

1. 接线位：`internal/portfolio/lifecycle/run.go` 周期拍挂
   `hunger.New(pool, cfg)`——Detect 先于 Respond，每拍至多
   响应一次（窗口判据内部已带）；
2. 节拍选择照 run.go 现有拍结构（scan 内挂或独立 ticker——
   与现有拍节奏对齐，禁新建 goroutine 野径）；
3. 失败隔离：Detect/Respond 败=log+不拖 lifecycle 主拍
   （T-0369 容错律：败必须自报 ops_event 升件行，非仅 stderr）；
4. 职卡/数据流同步一行（hungertick 入岗）。

## 验收锚

① lifecycle 进程内 hunger.Detect 每拍被调（桩测/日志实证）；
② 触发形构造（零提案窗/papar 池空桩）→ Respond 四步跑通→
   ops_event 饥饿响应+参数调整行落账；
③ 败径桩测：Detect 假败→主拍不炸+自报行落；
④ check_all 绿戳对口收口 SHA；镜像面件——并入下次发窗。

## 红线

- 不改 hunger 包本体语义（接线非重写）；
- 不动业主面键/凭证；
- 不动真单径。

## 结果回填

**实现摘要**（run=claude-t0378）：

1. **接线位与节拍**：`internal/portfolio/lifecycle/hunger.go`（新件，
   87 行——run.go 265 行近棘轮线，照包内 file-per-concern 惯例拆出，
   run.go 仅 Run 内 `go r.hungerLoop(ctx)` 一行启动+Runner.sensor 缝
   字段）。**独立 ticker（6h）非 scan 内挂**——勘查实拍：lifecycle scan
   纯事件驱动（events.ListenLoop ev_strategy/ev_eval），饥饿态=零提案零
   策略变=事件恰好静默，挂 scan 则检测器在最需跑时结构性失明；票面
   预留「或独立 ticker」径即此，SilentWatch 看门拍同构先例
   （`// sched-exempt-ok` 已锚：看门判定节拍非轮询拉取）。协程=Run 内
   BeatLoop 同款 ctx 界内拉起（非野径）。
2. **拍语义**（对形 tools/hungertick 拍壳）：Detect 先于 Respond 每拍
   至多一响（窗口判据 Detect 内部已带）；触发即落 ops_event `饥饿` 行
   （seat=决策席/ticket=T-0376——hunger.go hungerSeat/hungerTicket 同款
   口径，字面量本地复制未动 hunger 包本体）；非触发拍 log 一行饱食态
   （生产日志实证拍活性）。
3. **败隔离+容错必报**（票面第3条/T-0369）：Detect/Respond 败=log+
   ops_event `升件` 行（esc_class=写败/esc_scope=LIFECYCLE-HUNGER 判例
   键，判类册追记一行），note 前缀同窗 24h 至多一行（ops_esc.sh 同构
   NOT EXISTS 去重——同根因不暴）；自报行自身写败=静默退（升件不能再
   升，T-0369 结构性限度在案）。respond 败环级自报补 logStep 自身败情
   盲区（步级败情 hunger 包内已自落 `饥饿响应` 行）。
4. **hunger 包本体零动**（红线①）：git diff 实拍 hunger/ 目录零碰；
   业主面键/凭证零触、真单径（scan/openLive）零碰（红线②③）。

**测试证据**（exit 0 实拍，O_TEST_PG_DSN=5434 独立实例）：

- `go test ./internal/portfolio/lifecycle/ ./internal/portfolio/hunger/
  -count=1` → ok ×2——新增桩测四件（lifecycle/hunger_test.go）：
  ①TestHungerLoopDetectEveryBeat（锚①：独立节拍逐拍调 Detect 日志
  实证+非触发零误响+ctx 净退）②TestHungerTickTriggeredRunsRespondWithRows
  （锚②：净库零提案触发→饥饿行+四步饥饿响应行齐+休眠族复活翻位+
  参数调整行三列）③TestHungerTickDetectFailIsolatedSelfReported（锚③：
  Detect 假败→主拍不炸+respond 零误响+升件行落+同窗去重二拍不叠）
  ④TestHungerTickRespondFailSelfReported（锚③补面：Respond 假败环级
  自报行落）。
- `go build ./... && go vet ./... && gofmt -l internal/` → 全绿零输出。
- run.go 269 行（<300 棘轮线——接线块拆 hunger.go 防 check-lines 新超标）。

**验收锚对账**：

- ① Detect 每拍被调：PASS（TestHungerLoopDetectEveryBeat+饱食 log 行）
- ② 触发→四步+落账：PASS（TestHungerTickTriggeredRunsRespondWithRows）
- ③ 败径隔离+自报：PASS（TestHungerTickDetectFailIsolatedSelfReported+
  TestHungerTickRespondFailSelfReported）
- ④ check_all 绿戳：见收口行（决策席验戳时点对本 SHA）

**偏离说明**：

- 节拍取 6h（票面留白项）：窗=3 天尺度下检测时延 ≤6h 运营无关紧要，
  周期拍对形 ops_event 行量可控（持续饥饿态 20 行/天上限，对比小时拍
  120 行/天防 T-0360 周评审账面淹没）；var 锚出可测，生产值决策席可改。
- 败自报判例键=写败/LIFECYCLE-HUNGER（判类册「新类随写随生，本册追记」
  条款追记一行）——detect 属读败语义轻微拉伸，取 T-0369 自报类单键
  简净优先，scope 隔离判例面。
