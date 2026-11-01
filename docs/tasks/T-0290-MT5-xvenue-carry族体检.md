# T-0290 MT5/xvenue carry 族体检（T-0118 拆解·族票⑤）

| 字段 | 值 |
|---|---|
| 状态 | 收口待审 |
| 立项 | `[角色:决策]` @ 2026-11-01（T-0118 母票按机会族拆解首批） |
| 领取 | devin-施工席 @ 2026-11-01 |
| 优先级 | 中（跨 venue 面——能力账边界族） |
| 涉及面 | `fin_opp_eval(xvenue-carry)`、md 镜像 mt5_swap/carry_diff 信号、MT5 持仓/报价面、`fin_capability_surface` MT5 行 |
| 执行径 | 无头/交互皆可 |

## 背景

跨 venue carry 族（xvenue-carry 7d eval 仅 3 件）——T-0272 已复
mt5_swap/carry_diff 信号镜像双端同步（数据面在供），但评估产率
≈0。本票定性：信号在而无评估是模板缺席（文法未生成族）还是
执行侧无 venue 能力（capability 缺行）。

## 钦定界

- xvenue-carry eval 3 件解剖（标的/评分/未 proposed 因）。
- MT5 信号面体检：carry_diff/mt5_swap 行数+新鲜度（数据在案）。
- 能力面 gap：fin_capability_surface MT5/xvenue 行在否——
  「看见但无 executor」显形还是能力在而模板缺。
- 裁定建议：立模板（走 T-0120 文法化）/扩 universe/搁置——
  依据落票面。

## 验收标准

- [ ] eval 3 件解剖+信号面行数实证
- [ ] capability 覆盖判定（模板缺 vs 能力缺明判）
- [ ] 裁定建议落票面

## 范围外/禁线

- 不新建模板不动 capability_surface 不部署

## 结果回填（devin-施工席 2026-11-01）

**eval 解剖**：xvenue-carry 7d 仅 3 件全 rejected——净 APR 分布 -0.019~0.043（低于任何 proposed 阈——机会面如实无货非口径死）。

**信号面实证**：mt5_swap 3 行/carry_diff 2 行/xvenue_spread 3 行，max(ts)=11-01 02:55~03:41——T-0272 修复后镜像鲜活（PG-md↔SG 双端追平在案）。

**capability 判定**：fin_capability_surface 无 MT5/xvenue executor 行——**「看见但做不了」显形族**（信号在供、eval 在跑、executor 缺席）；模板亦缺（fin_opp_template 无 xvenue 行——eval 3 件系他族评估漏网非本族模板）。

**裁定建议**：**走 T-0120 文法化立模板**——依据：信号面就绪（镜像鲜活）+能力面明缺=正中文法化钦定场景「生成 candidate 模板+缺 executor 自动出需求卡」。搁置判据=低 regime（3 件 APR<5%——即使模板立了当前机会面也稀）；扩 universe 无意义（venue 内机会未见）。

**范围外确认**：模板零新建、capability_surface 零动、零部署。
