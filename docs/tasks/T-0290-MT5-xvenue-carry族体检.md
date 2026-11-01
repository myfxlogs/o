# T-0290 MT5/xvenue carry 族体检（T-0118 拆解·族票⑤）

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` @ 2026-11-01（T-0118 母票按机会族拆解首批） |
| 领取 | （session 标识 @ 时间；空=未领） |
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
