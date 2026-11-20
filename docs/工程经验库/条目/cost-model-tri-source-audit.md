---
schema_version: 2
id: cost-model-tri-source-audit
kind: map
risk: high
status: verified
keywords: [成本口径, 费率, fee, friction, 滑点, 对拍审计, eval, exec, 三源]
scope: [钱径, 成本模型, 参数面, 判据口径]
summary: 工单#25805（#25771 裁定① adopt-bounded）成本一致性审计一期——eval 估费/exec 实费/friction 校准三源逐字段对拍：单位三分（bps↔分数）、合流仅 funding 一脉、MT5 腿零费测=friction 校准对 mt5 域永缺、值漂实测（declared taker 5.0bps vs measured 1.76bps/leg）。纯审计件零码改
first_seen: 2026-11-20
last_verified: 2026-11-20
skills: [施工席]
evidence_refs:
  - type: code
    ref: internal/research/funding/eval.go:71-85,113-120（taker/maker 复合+CostAPR）
  - type: code
    ref: internal/research/dirmom/run.go:255（net_apr=gross−friction_rt·365/hold_days）
  - type: code
    ref: internal/research/gridfit/eval.go:62,73（fee_rt 每边名义费）
  - type: code
    ref: internal/trading/okxexec/order.go:80-108（ResolveOrderDetail→fee/feeCcy）
  - type: code
    ref: internal/trading/okxexec/bills.go:11,50-62（合约腿 fee 走 bills 域）
  - type: code
    ref: internal/trading/execution/flow_leg.go:118-141（legCash→fin_order_leg.fee，仅 USDT 触账）
  - type: code
    ref: internal/portfolio/lifecycle/friction.go:63-155（frictionCalibrate 30d 回灌）
  - type: code
    ref: internal/portfolio/lifecycle/opener_gate.go:95-108（T-0521 读侧合流 fee/4·slip/2）
---

# 成本口径三源对拍审计（一期——纯对拍，零 schema 零码改）

**裁定锚**：#25771 开源架构审计①「统一成本模型」adopt-bounded——
三口径并存实锤但不重建模型；一期只出对拍审计+差集表+二期建议。

## 三源结构

```
eval 估费（评估臂声明参数）          exec 实费（成交回执实算）        friction 校准（结算对账差）
research.*.{taker,maker,slip}_bps    okxexec/order ResolveOrderDetail  lifecycle/friction.go 6h拍
research.*.friction_rt / fee_rt      okxexec/bills（合约腿 fee 域）   →DMSet lifecycle.friction_*.okx
dcd 宣告 0                            mtexec 零费捕获                  opener_gate breakevenOK 消费
        │                                    │                              │
        └────────── 互不读同源 ───────────────┘                              │
              唯一合流点：opener_gate.go:100-108（funding 一脉）◄─────────────┘
```

## 差集表（逐字段对拍——每格 file:line 锚）

| 维度 | eval 估费 | exec 实费 | friction 校准 |
|---|---|---|---|
| 费率键形 | `research.funding.taker_fee_bps`（bps/单腿）/`maker_fee_bps`/`slippage_bps`/`fail_cost_bps`/`window_loss_bps`——run.go:25-29 | 无键——venue 回执实值 | `lifecycle.friction_{bps,fee_bps,slip_bps,at}.<venue>`——friction.go:152-160 |
| 单位 | **bps/万分之一**（funding）↔ **分数**（dirmom/dirtrend `friction_rt`、gridfit `fee_rt`）↔ 0（dcd declared）——**单位三分在案** | venue 原值 USD/计价币（okx fee 字段负值=成本） | bps/往返（fee+slip 合成分记） |
| 语义口径 | funding=单腿单边费×4=往返（eval.go:73,116）；dirmom/dirtrend=往返摊年化（run.go:255/eval.go:113）；gridfit=**每边名义费**（eval.go:62,73——买卖各计） | spot 腿 fee 原值记腿（flow_leg.go:122-124）；feeCcy≠USDT 不触现金账（execute.go:42,flow_leg.go:141）；合约腿走 bills 不消费本径（order.go:78-79） | Σ\|腿 fee\|/Σnotional×1e4（friction.go:115）+滑点=fin_slip_sample 均值×仓均腿数（:128-141）；1bps 地板（:143） |
| maker 分层 | funding 独有（eval.go:71-75 maker_fee_set 在册复合/缺=双 taker） | 实返自然含 maker/返佣符号 | taker/leg 数比注入口径行（:82-84，maker=post_only 判） |
| 滑点源 | funding 声明值（在册 0） | 不实测 | fin_slip_sample 实测（:129-136）；不足=5bps unmeasured 缓冲（:22-24,117-119） |
| 更新径 | 参数种/ops_event 进气道 DMSet | venue 成交回执实时 | frictionCalibrate 6h 拍 30d 已平仓窗，n<3 不覆写（:63-67,107） |
| 消费点 | funding breakevenOK（opener_gate.go:100-108 分量新鲜≤30d 替声明）/paper 扣点（friction.go:41-59） | legCash→fin_position 净额 | breakevenOK 读侧合流（唯一——他臂零消费） |
| venue 覆盖 | 全臂同参数（无 venue 分键——funding 参数全局） | okx 全腿；**mtexec 零费捕获**（mtexec.go 无 fee/commission 字段——MT5 腿 Fee=0 略，execute.go:42） | venue 分键；**mt5 域永缺**（fee NULL→unmeasured>0→整仓剔，:80-84 过滤——在册仅 .okx 键实证） |

## 在册值漂（2026-11-20 生产 app_config 实测）

| 键 | 在册值 | 对拍结论 |
|---|---|---|
| `research.funding.taker_fee_bps` | 5.0 | declared 单腿 5bps vs 实测 friction_fee_bps.okx=7.05bps **往返**→breakevenOK 合流推 TakerFeeBps=7.05/4≈**1.76bps**——声明值为实测 ~2.8×（保守向） |
| `research.funding.maker_fee_bps` | 0.58 | maker 分层仅 funding 一脉使用 |
| `research.funding.slippage_bps` | 0 | declared 0 vs 实测 friction_slip_bps.okx=0.165bps——声明面零滑点 |
| `research.dirmom/dirtrend.friction_rt` | 0.001（=10bps 往返） | vs 实测 friction_bps.okx=7.22bps（fee+slip）——1.4× 保守 |
| `research.gridfit.fee_rt` | 0.001（迁移播种=10bps/边） | vs 实测单腿 ~1.76bps——~5.7× 保守（生产键未种——fin_0215 待部署面） |
| `lifecycle.friction_bps.okx` | 7.2185 | 实测定盘（fee 7.05+slip 0.165，at=1795156413 新鲜） |
| `lifecycle.friction_*.mt5` | **键缺席** | MT5 腿零费测→unmeasured 剔→校准对该 venue 永不产出——兜 20bps 默认率 |

## 差集裁定（统一建议二期面——本票不实施）

| # | 差集 | 性质 | 二期建议 |
|---|---|---|---|
| D1 | **单位三分**：bps/边（funding）↔分数往返（dirmom/dirtrend）↔分数/边（gridfit）↔bps/往返（friction） | 口径层 | 读侧合流件统一归一 bps/往返中间表示——各臂声明参数照读、合流时换算（opener_gate fee/4·slip/2 已是此形先例） |
| D2 | 合流仅 funding 一脉——dirmom/dirtrend/gridfit eval 零消费 friction 实测 | 覆盖层 | opener_gate 分量替换模式推广到他臂判据读径（实测新鲜≤30d 则替声明值） |
| D3 | **MT5 腿零费捕获**——mtexec 不回传 commission/swap→leg fee NULL→friction 校准 mt5 域永缺 | 测量破口 | mtexec 订单回执补 commission/swap 字段捕获+leg 落账（工单#21129 okx 侧同款补法）——独立立票面 |
| D4 | feeCcy≠USDT 费不触现金账（execute.go:42 钉实） | 口径已知面 | friction 校准已按 px 折 USDT 归集（:81）——现金账不触但测量面已罩 ✓ 无破口 |
| D5 | paper 扣点 vs live 实费双径（chargePaperFriction 预支往返 vs legCash 实记） | 设计在案 | 不同源不同用（paper 定价 vs live 对账）——统一面只须共享 friction_bps 读径 ✓ 已罩 |
| D6 | 值漂：declared 全谱保守于实测（taker 5→1.76、fee_rt 10→~1.76） | 方向安全 | 保守向偏=判词假拒非假过——二期合流后自动消漂，无独立处置 |

## 总评

三源非「三份错」而是「三层事实」：declared（假设）/measured（venue 回执）/
calibrated（对账回灌）各有存在理由——裁定① 的「统一」应落在**读面合流**
（measured/calibrated 新鲜则替 declared）非模型重建。当前唯一合流点
opener_gate.go:100-108 证明模式可行；残面=D2 覆盖（他臂未接）+D3 测量破口
（MT5 域校准永缺——最实破口，等同 B4 级建议独立立票）。

证据锚：internal/research/funding/{run.go:25-29,eval.go:71-85,113-120}；
internal/research/{dirmom/run.go:52,255,dirtrend/eval.go:113,gridfit/eval.go:62,73,dcd/eval.go:126-127}；
internal/trading/okxexec/{order.go:80-108,bills.go:11,50-62}；
internal/trading/execution/{execute.go:42-43,flow_leg.go:118-141}；
internal/trading/mtexec/mtexec.go（零 fee 字段实证）；
internal/portfolio/lifecycle/{friction.go:19-25,41-59,63-155,opener_gate.go:95-108}；
生产 app_config 在册值（FIN_PG_DSN 实拉 2026-11-20）。
实证环境：origin/master 核读（2026-11-20）。
