# T-0297 funding-hedge 经济性根修——费用年化摊销＋翻面持续性＋最短持仓＋仓位数上限

| 字段 | 值 |
|---|---|
| 状态 | **待办** |
| 立项 | `[角色:管理]` devin-cli @ 2026-11-03（D-0043；CTO 事后会签） |
| 领取 | （空） |
| 优先级 | **最高**——D-0037 KPI 唯一主线；D-0042 11-15 止损日前须 paper 实证 |
| 涉及面 | `internal/research/funding/eval.go`（+`_test`）、`internal/research/funding/run.go`（参数键）、`internal/guard/guard.go`（+`_test`）、`internal/portfolio/lifecycle/opener.go`（开仓数闸）、新 migration（三键 seed）、o-research/o-core 重部署 |
| 承接 | T-0237 遗留（maker 径 probe 实证债）并入本票验收 |

## 病灶（实查在案，勿重复诊断）

**件① 费用未年化**：`eval.go` `cost := (feeBps+slippage+fail+window)/1e4; net := gross − cost`——
`gross` 是年化率，`cost` 是**一次性**往返分数，两者直接相减。持仓 1 天的 12bps 往返
真实成本≈43.8% APR，评估器按 0.12% 计；持仓 3 天≈14.6%。评估器系统性低估成本
数十倍→阈值形同虚设→开仓即亏费。D-0037 账面「费 3.05 vs 利息 1.08、1114 单」
即此病灶的表现。10-29 把 `min_net_apr` 抬到 0.20 是在口径错误下的补偿，根修后须回落。

**件② guard 单次翻面即平**：`guard.go:216-229` 最新 funding 与开仓方向异号且
|率|>`guard.flip_bps` 即 `funding_flip`。funding 在零附近抖动一次就平仓，再被评估器开回
——往返费白交。无「最短持仓」概念（`max_hold_hours` 只管超龄）。
（`funding_flip_preempt` 径已结构性死：OKX `nextFundingRate` 恒 `""`，T-0084 作废裁定留码不启——本票不依赖、不改。）

**件③ 无 live 仓位数上限**：risk 面有 `max_exposure_usd`/`max_notional_usd`/`max_venue_pct`，
无并发仓数闸；$10k 本金摊成十几条小仓＝每条都摊不薄固定摩擦。

## 钦定目标

1. **费用年化摊销**：新键 `research.funding.min_hold_days`（seed=3）。
   `cost_apr = cost_frac × 365 / min_hold_days`；`net = gross − cost_apr`。evidence 加
   `cost_apr`/`min_hold_days` 两字段。`research.funding.min_net_apr` seed 0.20→**0.06**
   （静态底；Earn 动态底 D-0006 照旧兜，`floor=max(静态, Earn)`）。
2. **翻面持续性**：新键 `guard.flip_persist_n`（seed=2）。`funding_flip` 须**连续 N 次
   已结算** funding 异号且超 `flip_bps` 才触发（读 `md_funding` 近 N 行）。
3. **最短持仓前禁平（仅限收益类触发）**：持仓年龄 < `min_hold_days` 时跳过 `funding_flip`
   /`max_hold`；**`delta_drift` 与 risk kill_switch 不受限**（风险闸永不让位）。
4. **live 仓位数上限**：新键 `risk.max_live_positions`（seed=3）。lifecycle opener 开仓前
   数 live 态 funding-hedge 仓，≥上限＝拒开（reason `max_live_positions`）。
5. **开仓时同口径**：`opener.go:118/161` 开时翻面预判改用持续性口径（N 次历史异号），
   与 guard 对齐。
6. 判据键缺＝fail-closed 跳过＋计数（沿用 guard 既有纪律）；**零代码字面量**，三键全走 config。

## 验收标准

- [ ] `eval_test`：同 rate/fee 下 `min_hold_days=1/3/30` 三档 net_apr 单调递增且数值＝手算；0.12% 费 1 天持仓 → rejected
- [ ] `guard_test`：一次异号不平、连续 2 次异号平；最短窗内 flip 不平但 delta_drift 照平
- [ ] opener 测试：live 仓数=3 时第 4 条拒开，reason 落 evidence
- [ ] migration seed 三键＋`min_net_apr` 改值，`cfg.Watch` 或重启生效径写明（T-0237 坑：o-research 仅 boot 快照）
- [ ] **30d 回放**：用 SG `md_funding` 近 30d 数据跑新旧口径对比——新口径下开仓次数、模拟往返费、模拟 funding 实收，表格落票面（目的＝证明「单数降、净利升」，非证明盈利）
- [ ] paper 7d：proposed→paper 仓最短持仓≥3d 实证，无最短窗内 flip 平仓
- [ ] maker 径 probe 实证（T-0237 债）：live 执行单 maker 成交比例＋真实 fee_bps 落票面
- [ ] 动真钱参数（min_net_apr/仓数）变更留前后对照＋出处 D-0043

## 范围外

- 不动发现面（md_signal 产出）、不碰 dcd/volsell 族；basis 族费用摊销归 T-0298
- 不改 risk 硬线（max_exposure/kill_switch）
- 不放量——live 放量由管理席按 paper 7d 实证另裁

## 参考

D-0043 / D-0037 / D-0006（Earn 动态底）/ T-0237 / T-0238（maker 径已并 master：maker.go、okx.go:68、eval_test TestMakerFeeSetSplit）/ T-0286（低 regime 如实细流）/ `skills/策略评估与仓位`

## 结果回填

（施工写）
