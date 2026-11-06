# T-0383 carry_diff 消费闸 120s 失明——xvenue 臂 DOA 根修

- 状态：施工中（2026-11-06 devin-t0383 领）
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

（施工写：实现摘要 / 测试证据 / 偏离说明）
