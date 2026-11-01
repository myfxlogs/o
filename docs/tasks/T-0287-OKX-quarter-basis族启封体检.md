# T-0287 OKX quarter/basis 族启封体检（T-0118 拆解·族票②）

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` @ 2026-11-01（T-0118 母票按机会族拆解首批） |
| 领取 | （session 标识 @ 时间；空=未领） |
| 优先级 | 中→高 |
| 涉及面 | `fin_opp_template(quarter_basis_long/quarter_basis_short/perp_quarter_spread)`、md 季货行情面、`fin_opp_eval`、`fin_proposal` |
| 执行径 | 无头/交互皆可 |

## 背景

basis/quarter 套系三模板全 `enabled=f`（7d eval 产出仅 11 件——
quarter_basis_long 4/perp_quarter_spread 4/quarter_basis_short 3）。
armed 提案实拍 BTC-USD-270326 basis_quarter/perp_quarter_spread 在架
（10-31）——评估器在跑但模板禁=晋升无路。启封体检=定性本族
「能不能开、该开哪个模板、数据面够不够」。

## 钦定界

- 三模板禁因回追：谁/何时/为何 disabled（ops_event/git log 实证）。
- 数据面体检：季度期货行情在 md 面覆盖度（universe 放开后
  T-0122 依赖在否——当季/次季合约报价实读率）。
- eval 产出解剖：11 件评分分布+未 proposed 原因（机会真缺 vs
  口径过严 vs 数据缺）。
- 启封建议：三模板逐一「启用/保禁/退役」裁定建议+启用者验收口径。

## 验收标准

- [ ] 三模板禁因台账（来源实证）
- [ ] 季货行情覆盖率实证（md 表行数/新鲜度）
- [ ] eval 产出解剖表+未 proposed 归因
- [ ] 逐模板启封建议+启用验收口径落票面

## 范围外/禁线

- 不直接翻 enabled（启封另派）不动判据参数不部署
