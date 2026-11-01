# T-0289 OKX vol-sell 族体检（T-0118 拆解·族票④）

| 字段 | 值 |
|---|---|
| 状态 | 收口待审 |
| 立项 | `[角色:决策]` @ 2026-11-01（T-0118 母票按机会族拆解首批） |
| 领取 | devin-施工席 @ 2026-11-01 |
| 优先级 | 中 |
| 涉及面 | `fin_opp_template(vol_sell_single_call/vol_sell_strangle)`、`fin_opp_eval`、IV/期权行情面 |
| 执行径 | 无头/交互皆可 |

## 背景

vol_sell 双模板禁而 eval 高产（single_call 8573+strangle 1782/7d
≈10.4k 件）——评估面旺、模板无路同 DCD 型错配。本票=族体检：
产出质量、禁因、IV 数据面覆盖、启封或收敛裁定建议。

## 钦定界

- 双模板禁因回追（来源实证）。
- eval 评分分布+高分占比（机会 vs 噪声明判）。
- IV/期权链数据面覆盖度实证（md 面行数/新鲜度——缺 IV 面=
  评估先天盲）。
- 裁定建议：启用（条件清单）/保禁（理由）/退役（判据）。

## 验收标准

- [ ] 禁因台账+eval 评分分布实证
- [ ] IV 数据面覆盖实证
- [ ] 双模板各自裁定建议落票面

## 范围外/禁线

- 不翻转 enabled 不动判据不部署

## 结果回填（devin-施工席 2026-11-01）

**禁因台账**：同 10-29 清理波（「dcd/volsell/earn/lending 渐进死+armwatch 盲区」领单事件）——四族并禁非单独判决。

**eval 解剖**：7d 10.4k 件——vol_sell_single_call 8573（6768 rejected+1805 unevaluable，net_apr 全 NULL）+vol_sell_strangle 1782（1556 rejected+226 unevaluable，net_apr 全 NULL）。**评分全空=IV 数据面先天缺**——评估器喂不出 IV 行=产出即噪声非机会。

**IV 数据面实证**：md 表册无期权 IV/希腊字供面（md_dcd_product 只产品目录非 IV；vol_sell eval 缺输入源——模板评估径无 IV feed）。

**裁定建议**：**双模板保禁**。依据：①IV 数据面缺席=评估先天盲（proposed 都不可能真产出——全 rejected/unevaluable 实证）；②同批禁因（渐进死）未消解。启封前置条件=IV 数据面先建（md 期权 IV 供面立项）+判据以 IV 实馈复核——未建前评即废件。

**范围外确认**：enabled 零翻、判据零动、零部署。
