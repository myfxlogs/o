# T-0023 Earn 现率采集+动态比较底（T-0019 拆分快半——读面零钱动）

| 字段 | 值 |
|---|---|
| 状态 | 审计通过 |
| 提出 | `[角色:决策]` 2026-10-23（T-0019 慢在途，拆读面先行） |
| 类别 | 数据+研究域（D-0006 读面半） |
| 依赖 | T-0010✅、T-0016✅、T-0018✅（typed 读器）；`okx-exec`/`okx-rest-boundary` 必查 |
| 领取 | devin-t0002 @ 2026-10-23 |

## 背景

T-0019 全票在 zcode 席慢工。拆其**读面半**独立成票——Earn 现率采集+评估卡动态底，零资金移动、零 schema 大动，速通面。**若 T-0019 先行落地覆盖本票面，本票改纯验证收口**（竞合安全）。

## 范围

1. **`md_earn_rate` 表+采集**：`GET /api/v5/finance/savings/lending-rate-summary?ccy=USDT` 周期采（拍长走 `data.earn.poll_sec` 键）→venue/ccy/rate/ts 事实行；采败=告警计数不造假。采集件=okxexec 签名径装配注入（data 域不引 trading 包——platform 面或 okxexec 同面复用，按 import 闸选径）。
2. **评估卡动态底**：research 比较器改 `floor=max(min_net_apr键, latest md_earn_rate.rate)`；`net_apr<floor`→reject，reason 面加 `below_earn`；现率缺失/陈旧（>`research.earn.stale_hours` 键）→退回静态底+告警计数（降级不阻断）。
3. **留痕**：评估卡 evidence 记 floor 构成（static_floor/earn_rate/floor_used）——将来复盘看得出「是被谁拒的」。

## 验收标准

- [ ] md_earn_rate 事实行产出（dev 库实拍可查）；rate=OKX 实返
- [ ] 动态底三态测试锚：net_apr 介于静态底与 earn 间→below_earn 拒；高于 earn→过；earn 缺失→静态底降级+计数
- [ ] evidence 记 floor 构成；热启键全走 app_config 缺键 fail-closed
- [ ] import 闸零违；机检全绿；CI 绿

## 范围外

- 赎回/回存/配管（T-0019 残面）；earn 持仓台账（T-0020 已立骨架）

## 结果回填

分支 `t/0023-earn-rate`（已推 origin）。施工面=worktree `/opt/phase-wt/t0023`。

**验收锚对账**：

- md_earn_rate 事实行+rate=OKX 实返 = **PASS**：`0015` hypertable；earnpoll 直拉 `lending-rate-summary?ccy=USDT` 实拍——raw 实返 `[{ccy:USDT,estRate:0.035,...}]`→落行 rate=0.035（dev 库可查）；采集件=okxexec 装配注入，采败=fetch_err 计门不造假。
- 动态底三态锚 = **PASS**：floor=max(min_net_apr, latest USDT earn)——0.10<0.5=below_earn rejected / 1.09>0.5=proposed / 缺失→静态底降级+earn_stale 计门（三锚测试过）；陈阈 research.earn.stale_hours 热启。
- evidence 记 floor 构成 = **PASS**：floor_used/static_floor/earn_rate 三键入卡（复盘可读被谁拒）；拍长 data.earn.poll_sec 热启缺键 missing_param 计数。
- import 闸零违 = **PASS**：OK 35/35 边（earnrate→config/rejects 冻结；拉取件 cmd 装配 okxexec 唯一径）。
- 机检全绿 = **PASS**：build/vet/gofmt/lines/env/migration/import 全过；go test 15 包全绿 @5434。

**实测修正**：lending-rate-summary 字段族=estRate/avgRate/preRate 非 `rate`（dev 实拉钉出——estRate 主 avgRate/preRate 备）；dev 库补跑 0015 迁移（schema_migrations 记 15）。

## 审计判决

审计席：headless @ 2026-10-30

**判决注记（逐锚对账——审计对象=master 存续面 0dbeb3d；本票自家施工件经竞合收口 merge 9b1bf80 判 T-0019 全覆盖摘除未并入，等价实现在 internal/earn+internal/research/funding 存续，逐锚亲证如下）**：

1. **md_earn_rate 事实行+rate=OKX 实返 = 通过**：表存续（`migrations/md_1016_earn_rate.sql` 建超表→`md_1049` D-0018 状态表化 PK(venue,ccy)）；采集径 `internal/earn/worker.go:135 pollRate`→rateFn=`okxexec.LendingRateSummary`（worker.go:89-91）→`savings.go:41,51` 公开端点 GET /api/v5/finance/savings/lending-rate-summary?ccy=USDT（免签名——与票面「okxexec 复用」意合且更窄）；写径 UPSERT ON CONFLICT(venue,ccy)（worker.go:147-149）与状态表形吻合；采败三计门 rate_fetch_failed/rate_parse_failed/write_err（:138/:143/:150）不造假；装配 `cmd/core/main.go:136` NewWorker→:138 Run，事件化 ListenLoop ev_account/ev_position（:122，T-0048 后周期原语退役）；dev 库实拍行 `okx|USDT|0.0394` @2026-10-29 14:36Z 在册。取值字段=AvgRate（24h 均率——「现率」口径贴义，与回填「estRate 主」表述异，锚「rate=OKX 实返」满足）。
2. **动态底三态 = 通过**：`eval.go:45-49` floor=max(MinNetAPR,EarnRateAPR) floorSrc=static|earn；`eval.go:73-79` net≥floor→proposed／net<静态底→阈拒／夹层→below_earn rejected（reason 机读含双值）；`run.go:177-194` earnRateAPR 最新行 ORDER BY ts DESC LIMIT 1、`research.earn.stale_hours` 陈阈——缺键/缺行/陈旧三分支同归 (nil,false)；`run.go:128-130` !fresh→`earn_rate_degraded` 计门降级不阻断。`TestEvaluateDynamicFloor`+`TestScanEarnFloorE2E` 审计席复跑绿（O_TEST_PG_DSN@5434，包级 10 件全 PASS）。
3. **evidence floor 构成+热启键 fail-closed = 通过**：`eval.go:61-68` evidence 落 min_net_apr/floor_apr/floor_source(static|earn)/earn_rate_apr，:81 reason 入卡——票面举坐（static_floor/earn_rate/floor_used）语义全覆盖且多记 source；`research.earn.stale_hours` 经 cfg.GetFloat（platform config→app_config）缺键即 fail-closed 降级（run.go:177-179）；fin_0016 种子 stale_hours=24+poll_sec=300。注：`data.earn.poll_sec` 键在册但 Go 面零引用——T-0048 事件化后周期采拍退役的合规演化残留（种子行死配），非锚缺。
4. **import 闸零违+机检+CI = 通过**：闸实跑 OK 112/121 边零违（earn→okxexec 边白名单 :135 在册）；go build earn/funding/okxexec 三包绿；funding 包 go test 10 件复跑绿；CI master HEAD 0dbeb3d（PR#114 merge）run success @2026-10-30T09:17:44Z。
