# T-0383 carry_diff 消费闸 120s 失明——xvenue 臂 DOA 根修

- 状态：待验（2026-11-06 devin-t0383 收口移交决策席）
- 领取：常径机派；实工领取=devin-t0383 @ 2026-11-06（worktree /opt/o-wt/t0383）
- 优先级：**P0 钱径**——评估臂对现有信号失明=觅食面直接断粮
- 收口移交归：决策席

## 病灶（实证锚）

- 写径 T-0164 血统闸：carry_diff.ts=least(fundingTs,swapTs)
  （xvenue.go ~131 行注释自白——mt5_swap 4h 拍面陈→行随陈）；
- 消费闸 xcarry.go ~33：carry_diff 读 `ts>now()-interval
  '120 seconds'`——慢源信号一写即陈，闸恒视缺席；
- 实证：fin_opp_eval template='xvenue-carry' 全史 3 行（10-26
  /10-29/10-30 皆 rejected——且 3 次恰撞 swap 刚写<120s 运
  气窗）；当前 md_signal carry_diff 值 BTC -515bps / ETH
  +455bps vs 阈 250bps——机会在桌面评估臂看不见；
- 口径矛盾：读侧 xvenue.go:96 mt5_swap 闸已 8h，同族消费
  侧 120s——同一信号两种鲜度标准。

## 范围

1. `xvenueCarryScan` 消费闸改**源节拍感知**——carry_diff
   鲜度界=上游最慢源 mt5_swap 4h 拍+安全边（建议 8h 与读侧
   对齐，或 meta 内 ann_ok/cadence 字段机读派生——禁拍头
   魔法数,派生径注注释引证）；
2. T-0164 血统闸语义不破——ts 仍=最旧上游观测（改的是
   消费侧判据不是写侧戳）；
3. 同步查同病：全仓 `interval '120 seconds'` 消费闸逐一
   对源节拍——其它慢源信号同病灶一并修（gapscan/basis
   等族列清单,凡源拍>闸值=同案）；
4. 修后回归：carry_diff 现值(BTC -515/ETH +455)在消费面
   应可见——桩测重放真实信号行→评估产出断言。

## 验收锚

① carry_diff 陈行（ts=2h 前）桩测：消费臂见信号+评估
   落行（修前零行修后有行——翻转断言）；
② 全仓 120s 闸审计清单落票（哪些改哪些不改+理由）；
③ T-0164 血统语义断言不破（ts 仍最旧上游）；
④ 既有桩测回归绿+check_all 全链绿戳对口；
⑤ 镜像面件并入发窗——合并后 xvenue 提案若出,lifecycle
   admitXvenuePair 既有径自动开纸面户。

## 红线

- 不改信号写侧 ts 语义（血统闸不破）；
- 不降阈（250bps carry_diff 阈照旧——修的是可见性非放宽）；
- 慢源数据陈旧风险评估随行（swap 4h 陈=决策材料陈,注释
  注明残险）；
- 不动真单径。

## 结果回填

实现摘要（devin-t0383 @ 2026-11-06，分支 t/0383-xcarry-gate-cadence）：

- **消费闸源节拍感知化**（范围①）：
  - `hypothesis/xcarry.go` xvenueCarryScan carry_diff 闸 `120s→8h`——
    界=最慢源 mt5_swap 4h 拍（`mt5/bridge.go swapEvery`）×2 安全边，
    与读侧闸同根对齐（`sigdist/xvenue.go` mt5_swap 8h 闸、
    `stakecarry.go` 同、`cmd/core/ops_face.go` 巡检 warn5h/dead9h
    在案）——派生径注释引证，非魔法数；
  - `sigdist/run.go` `Fresh`/`freshByKind` 种类自适闸扩
    `carry_diff→8h`（CASE kind 派生——T-0199 funding_view 三巡
    同形制四巡；同一信号两种鲜度标准的口径矛盾根修）；
- **写侧零动**（范围②/红线）：carry_diff ts 仍=least(fundingTs,swapTs)
  ——改的是消费判据非写侧戳；xvenue.go 写侧注释更新为现状语义；
- `docs/数据流.md` 消费闸口径同步（链路语义变更票同 commit）；
- 残险注释随行（红线）：swap 年化材料最陈≈4h 源拍固有——评估读源
  快照非决策刻现值，注释钉于 xcarry.go 闸上。

测试证据：

- **翻转断言** `TestXvenueCarryStaleVisible`（xcarry_test.go）：
  carry_diff ts=2h 陈行→xvenueCarryScan→fin_opp_eval 落行
  （rejected/capacity_missing 如实）；ts=9h 超界行→零评估行（闸未拆）。
  修前 120s 闸对 2h 陈行恒滤=零行——翻转成立（锚①）；
- `TestXvenueLineage` 尾锚翻转：`Fresh(carry_diff@2h)` 现可见、
  `@9h` 缺席；血统戳断言（ts=最旧上游 2h 前）原样保留 PASS（锚③）；
- `go test ./internal/research/sigdist ./internal/research/hypothesis`
  全绿（TestXvenue*/TestFresh*/TestStakeCarry*/TestXvenueCarry*）；
- check_all 全链绿戳对口（收口 commit 同拍，锚④）。

全仓 `interval '120 seconds'` 消费闸审计清单（锚②——逐一对源拍）：

| 闸位 | 信号 | 源拍 | 处置 |
|---|---|---|---|
| hypothesis/xcarry.go carry_diff 扫描闸 | carry_diff | ts=最旧上游，最慢源 mt5_swap **4h** | **同病→修 8h** |
| sigdist/run.go Fresh 通用闸 | 全 kind | carry_diff 经此可达（桩测实达；SurfaceSignals 映射当前不入该 kind） | **同病→CASE 自适 8h** |
| sigdist/run.go freshByKind 通用闸 | 全 kind | 同上（闸面同源病） | **同修——与 Fresh 同闸语义** |
| hypothesis/xcarry.go spreadAmortBPS/spreadVal | xvenue_spread | ts=最旧上游（mark/tick 皆秒级流，且写侧双 120s 闸内才产） | 不配——120s 即「上游活」判据 |
| hypothesis/xcarry.go 容量 depth5 读 | depth5 | okxws books5 WS 秒级帧 | 不配 |
| hypothesis/signals.go ScanSignals | depth5 / funding_view | depth5 秒级；funding_view 已当值期自适 | 不配 |
| sigdist/xvenue.go mark/tick 读 | md_mark_index / md_ticks mt5 | okxws WS / mt5 OnQuote 皆秒级流 | 不配（写侧上游闸，语义正确） |
| projector_test.go:460 | 夹具字面 | n/a | 不动 |
| 文档面（done/T-0199 票文/本票） | n/a | n/a | 不动 |

派生信号侧复：basis_ann（ts=最旧上游 mark/index-tickers 秒级 WS）/
xvenue_spread（同上）/funding_view（当值期自适）——源拍皆≤闸值，
唯 carry_diff 慢源同病，已修。

偏离说明：

- Fresh/freshByKind 产线径当前无 carry_diff 路由（SurfaceSignals 面→
  kind 映射不含该 kind）——同修属同病灶根治+口径矛盾钦定项（「同一
  信号两种鲜度标准」），非范围外扩；
- 「meta ann_ok/cadence 机读派生」未取：mt5_swap meta 无 cadence 字段
  （ann_ok 仅表换算成败），采票面主建议=读侧 8h 对齐；
- 审计观察项（未动，备决策席知）：xcarry.go 容量件 `md_ticks mt5` mid
  读无 ts 闸——非 120s 字面闸，秒级流残险低；
- 红线全守：阈 250/300bps 未动、写侧 ts 语义未动、真单径未碰；
- 坑处置随行：新坑条目 `条目/signal-fresh-gate-source-cadence.md`
  入库+kbgen 重建索引；伴生量表 `xvenue_carry_eval_zero`（断供类）
  注册 scripts/meta/gauges.tsv（照 stake_carry_eval_zero 同形——
  评估臂再失明自动升件）；
- 锚⑤ 移交注记：镜像面件（Go 码）入发窗由决策席按阻塞集判据合并；
  合并后 xvenue 提案若出，lifecycle admitXvenuePair 既有径自动开纸面户。
