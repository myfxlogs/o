# T-0297 funding-hedge 经济性根修——费用往返口径＋年化摊销＋开闸同源

| 字段 | 值 |
|---|---|
| 状态 | **已批**（CTO 会签 2026-11-03——病灶核实后范围修订，见「CTO 会签勘正」；派单序：T-0299 清带宽后派） |
| 立项 | `[角色:管理]` devin-cli @ 2026-11-03（D-0043）；会签 `[角色:决策]` devin @ 2026-11-03 |
| 领取 | （空） |
| 优先级 | **最高**——D-0037 KPI 唯一主线；D-0042 11-15 止损日前须 paper 实证 |
| 涉及面 | `internal/research/funding/eval.go`（+`_test`）、`internal/research/funding/run.go`（参数键）、`internal/portfolio/lifecycle/opener.go` breakevenOK（+`_test`）、新 migration（`min_net_apr` seed 改值）、SG `app_config` 改值、o-research/o-core 重部署 |
| 承接 | T-0237 遗留（maker 径 probe 实证＋真实费率档位）并入本票验收 |

## CTO 会签勘正（2026-11-03，SG 生产取证＋码面亲证——取代原「病灶」段的件②③）

原票面三件病灶，逐件核实结果：

**件① 费用未年化——成立（码面亲证）。** `eval.go:58-59`
`cost := (feeBps+slippage+fail+window)/1e4; net := gross − cost`——`gross` 是年化率，
`cost` 是一次性分数，直接相减。SG 现值（maker 键缺→双 taker）cost=19bps=0.0019：
按 3 天持仓真实年化成本 0.0019×365/3=**23.1%**，评估器当 0.19% 算，低估 ~120 倍。
且 19bps 本身还不是往返：fee 项 `2×taker` 只计一向两腿、slippage 只计一次——
四次成交（开两腿+平两腿）的往返口径应为 `2×feeBps + 2×slippage + fail + window`
=32bps（与 opener `breakevenOK` 的 `4×(taker+slip)=32bps` 才对得上）。

**件② 「guard 单次翻面即平→往返费白交」——不成立（生产取证）。** SG
`fin_position ⋈ fin_guard_close` 全量 22 仓：**trigger_kind 全部=delta_drift，funding_flip=0**。
`guard.flip_bps=5` 意味着 funding 须低于 −0.05%/8h（≈ −55% 年化）才触发——这不是
「零附近抖动」。翻面持续性（`flip_persist_n`）解决的是一个从未发生过的问题，**撤出本票**。

**件③ 「无最短持仓→磨损」——不成立，且方案有害。** 22 仓平均持仓 1.2h、21 仓 <1h
（多数开平同一分钟），全由 delta_drift 杀——根因是 10-24/25 上线期执行面缺陷
（量纲误读/陈 tick 假失衡/基线缺位，已分别由 T-0064/T-0107/T-0128 根修；开平振荡保险丝
T-0103 已落）。「最短持仓前禁平」是沉没成本谬误：费已付出，禁止离场收不回费，只会在
funding 真翻负时多付负 funding。**撤出本票**。风险闸（delta_drift/kill_switch）本就不可让位。

**「1114 张单磨光钱」——叙事勘正。** `fin_execution` 1114 行=failed 689＋ambiguous 380
＋closed 42＋其他 3；真实成交只有 42 次执行（≈21 个 $50 往返）。费 3.05 来自这 21 个
往返，不来自阈值放薄利单——这些仓没活到第一次 funding 结算（funding_settle 全账仅 4 行
+0.002），任何阈值都救不了。1069 次 failed/ambiguous 是 10-24~10-28 执行面故障，零费。

**`min_net_apr` 0.20 的性质。** T-0237 抬阈的算式「20% APR≈5.5bps/日，72h≈16bps>12bps
摩擦」本身就是按 72h 摊销的往返口径——0.20 不是「错误口径下的硬补」，而是在评估器
不会摊销的情况下，把摊销后的门槛手工折算进了静态阈。根修后评估器自己摊销，静态阈
回落到 0.06 是口径归位，不是放松（见下「数字对照」）。

**件④ `risk.max_live_positions`——缓议。** `risk.max_exposure_usd=500` 已把 $50 probe
封顶 10 仓；无实证表明仓数是病灶。不入本票，留 D-0043 待议。

### 数字对照（SG 30d `md_funding` 12,508 观测/511 inst，Earn 现率 4%）

| 口径 | 开仓所需 funding/8h | 30d 观测占比 |
|---|---|---|
| 现状：gross−0.0019 ≥ 0.20 | ≥1.84bps | 4.1%（513） |
| 现状 opener breakevenOK：9 结算 carry ≥ 32bps×1.5 | ≥5.33bps | 1.7%（208） |
| 根修后：gross − 32bps×365/3 ≥ 0.06（H=3d） | ≥4.1bps | ~2% |

BTC/ETH/SOL 30d 中位 funding ≈0.7–1.0bps/8h（≈8–11% 年化毛收）。结论：**按 taker 费
＋3 天持仓期，OKX funding-hedge 在当前 regime 下几乎不该开仓——这是如实，不是缺陷。**
真正的杠杆是两件：持仓期拉长（guard 已有「trailing 24h 实收>0 免超龄」豁免，H 实际可
>3d）与 maker 费（T-0238 maker 径已并 master，SG `research.funding.maker_fee_bps` 键
**缺位**→评估仍按双 taker）。两者都在本票验收锚内度量，不在本票放量。

## 钦定目标（修订后）

1. **费用往返口径＋年化摊销**（`eval.go`）：
   `cost_rt_bps = 2×feeBps + 2×SlippageBps + FailCostBps + WindowLossBps`（feeBps 沿
   T-0247 分层：maker 在册=taker+maker，缺=2×taker）；
   `hold_days = guard.max_hold_hours / 24`（**复用既有键，不新增 `min_hold_days`**——与
   opener breakevenOK 同一时间地平线，一源）；
   `cost_apr = cost_rt_bps/1e4 × 365 / hold_days`；`net = gross − cost_apr`。
   `Params` 加 `HoldDays float64`；`run.go` paramKeys 加 `guard.max_hold_hours`，键缺=
   fail-closed missing_params（沿用）。evidence 加 `cost_rt_bps`/`hold_days`/`cost_apr`。
2. **开闸同源**（`opener.go` breakevenOK）：摩擦与年化改调 `funding` 包导出的同一函数
   （导出 `RoundTripBps(p Params) float64` 与 `CostAPR(p Params) float64`），删 opener
   内 `4*(taker+slip)` 本地算式；`guard.breakeven_safety`（缺省 1.5）保留作安全系数。
   拒因 evidence 字段改为 `cost_apr`/`gross_apr`/`safety`。自检 I：提案卡展示的 net_apr
   与开闸判据取同一函数。
3. **阈值归位**：`research.funding.min_net_apr` 0.20→**0.06**——新 migration seed
   `ON CONFLICT DO UPDATE`（现 fin_0006 为 DO NOTHING，SG 现值 0.20 不会被 seed 覆盖，
   须显式 UPDATE 径）；Earn 动态底 `floor=max(静态, Earn)` 不变。生效径：o-research
   仅 boot 快照（T-0237 坑）——部署即生效，票面写明重启锚。
4. **零代码字面量**：所有系数走 config；`hold_days` 由 `guard.max_hold_hours` 派生，
   不另立键。

## 验收标准

- [ ] `eval_test`：固定 rate=0.0003、Params 如 TestMakerFeeSetSplit 基线，`HoldDays`
      =1/3/30 三档 NetAPR 单调递增且与手算相等（1d：cost_apr=0.0032×365=1.168；
      3d：0.3893；30d：0.0389）；rate=0.0001（≈11% 毛收）、HoldDays=3 → rejected
      且 reason 含 `net_apr`；evidence 含 `cost_rt_bps=32`/`hold_days`/`cost_apr`。
- [ ] `eval_test` 既有 TestMakerFeeSetSplit/TestEvaluateDynamicFloor 按新口径改期望值
      （不删锚），`HoldDays=0` 为 missing_params（run.go 测试）。
- [ ] `opener_test`：breakevenOK 对同一 Params 与 `funding.CostAPR` 同值；删本地算式后
      测试红→绿（对抗性：改 `RoundTripBps` 系数 opener 测试必红）。
- [ ] migration：`min_net_apr` 0.20→0.06 显式 UPDATE；部署后 SG `app_config` 实查
      =0.06 且 fin_opp_eval evidence 带 `cost_apr`。
- [ ] **30d 回放**（SQL 由决策席出，施工只跑）：用 SG `md_funding` 近 30d 逐 inst 逐结算
      跑新旧口径——新口径 proposed 次数/旧口径 proposed 次数/对应 funding 分位，表格
      落票面（目的＝证明「口径对了」，不证明盈利）。
- [ ] paper 7d：新口径下 proposed→paper 仓统计（数量、持仓时长、模拟 funding 实收 vs
      模拟往返费）——零开仓也是合格结果，如实落票。
- [ ] maker 径 probe 实证（T-0237 债）：OKX `/account/trade-fee` 真实 taker/maker 档位
      （spot 与 swap 分列）落票；`research.funding.maker_fee_bps` 是否入册由数据裁。
- [ ] 动真钱参数变更留前后对照＋出处 D-0043＋本会签段。

## 范围外

- 不动 guard（翻面持续性/最短持仓两件已撤，理由见会签段）；不加 `risk.max_live_positions`
- 不动发现面（md_signal 产出）、不碰 dcd/volsell 族；basis 族费用摊销归 T-0298
- 不改 risk 硬线（max_exposure/kill_switch）
- 不放量——live 放量由管理席按 paper 7d 实证另裁

## 参考

D-0043（含 CTO 会签注记）/ D-0037 / D-0006（Earn 动态底）/ T-0237（阈 0.20 算式）/
T-0238（maker 径）/ T-0247（maker 键分层）/ T-0111（breakevenOK）/ T-0064·T-0107·T-0128
（delta_drift 误杀三修）/ `skills/策略评估与仓位` /
工程经验库 `loss-attribution-read-guard-trigger-first`

## 决策席 30d 回放 SQL 与预跑（验收锚⑤用料——施工部署后 SG 复跑同式贴表）

```sql
-- 参数出处（SG app_config 现值）：taker 5.0、slip 3.0、fail 2.0、window 4.0、
-- settles 3、max_hold 72h→hold_days 3、maker 键缺→feeBps=2×taker=10；
-- cost_rt_bps=2×10+2×3+2+4=32 → cost_apr=0.0032×365/3=0.389333；
-- 旧口径 cost=19bps 直减、阈 0.20；新口径 floor=max(0.06, Earn 现率)。
-- 方向面：rate>0 才可开（rate<0=现货做空腿无支撑 rejected），负率不计入 pass。
SELECT count(*) FILTER (WHERE funding_rate > 0) pos_obs,
 count(*) FILTER (WHERE funding_rate > 0 AND funding_rate*3*365 - 0.0019 >= 0.20) old_pass_pos,
 count(*) FILTER (WHERE funding_rate > 0 AND funding_rate*3*365 - 0.389333
   >= GREATEST(0.06,(SELECT rate FROM md_earn_rate WHERE venue='okx' AND ccy='USDT'
   ORDER BY ts DESC LIMIT 1))) new_pass_pos,
 count(DISTINCT inst_id) FILTER (WHERE funding_rate > 0 AND funding_rate*3*365 - 0.389333
   >= GREATEST(0.06,(SELECT rate FROM md_earn_rate WHERE venue='okx' AND ccy='USDT'
   ORDER BY ts DESC LIMIT 1))) new_pass_insts
FROM md_funding WHERE ts > now() - interval '30 days';
-- 逐 inst 榜：同窗 DISTINCT ON (inst_id) 最新行 × 新口径 net，附 spot 腿核（下文注意项③）
```

**预跑实数（决策席 2026-11-03 SG，只读）**：30d 12,508 观测中 pos=7,320；
旧口径 pass=511（7.0%）｜新口径 pass=259（3.5%）跨 75 inst。max 观测 100bps。

**注意项（贴表时同列）**：
①新口径过阈榜头部是**股权指数永续**（BWET/SHEIN/ADBE/US100…funding 极高但
  无现货腿）——`spotLegOK` 会拒，回放表须加一列「spot 腿在册？」（对照
  `fin_venue_untradeable` + OKX instruments SPOT 目录），过阈且有腿才是真候选；
②观测期正处低 regime（BTC/ETH 中位 ≈0.5–1.0bps）——pass 占比是 regime 函数非口径好坏；
③`md_funding` 30d 覆盖率依采集面（主流对自 10-30 断流过一次），表注观测窗。

## 结果回填

（施工写）
