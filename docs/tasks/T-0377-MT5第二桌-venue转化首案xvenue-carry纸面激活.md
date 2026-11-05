# T-0377 MT5 第二桌——venue 转化首案 xvenue-carry 纸面激活

- 状态：已批（2026-11-05 业主令「派出来」——缺陷判词 P0 对治：
  单 venue×单 alpha 存在性风险；MT5 真户 EXR 3001157 业主
  10-24 已授权交易，零新凭证）
- 领取：业主手动外派 @ 2026-11-05（外部工人 run——提示词
  docs/prompts/T-0377-O-Dev施工提示词.md，DM 勿自动领）
- 优先级：P1——战略缺陷对治（venue 转化首案）
- 收口移交归：COS（collab 写面判据窄口豁免——手动外派件）

## 背景与依据

- 判词（11-05 实账）：34 live 全 OKX funding-hedge=单 alpha 单
  venue；产出年化 ~2.9% < mandate 8%；侦察臂 10 候选转化=0。
- 机制在案：T-0144 建 `xvenue-carry` 主从双腿 paper 户
  （`internal/portfolio/lifecycle/xvenuepair.go`——主 okx perp
  收付 funding 端+副 mt5 CFD swap 端，向取反对冲）；映射源
  `app_config bridge.xvenue.map`；mt5_swap 信号快照建行时锁。
- MT5 面能力账：skills/mt5-* 五件全谱实测在案（quotes/history/
  account/orders-streams/trading-mutating）。
- **本票只到 paper 结算**——probe/live 迁移归生命周期判据+
  委员会闸，不开真单径。

## 范围

1. 链路实查：`bridge.xvenue.map` 现值与覆盖品种、mt5_swap 信号
   供径是否活、admitXvenuePair 触发位（谁调——评估提案径还是
   手工）、paper 结算对 mt5 腿的 funding/swap 记账口是否通；
2. 首案激活：选 1-2 个双所同品种（如 XAUUSD/BTC 主链映射内）
   开 xvenue-carry paper 双腿户——机读落地（fin_strategy 主从
   两行+fin_paper_fill 开流）；
3. 断链补焊：若查出任一环节断供（map 缺品种/mt5_swap 信号死/
   触发位无人调），在票面域内根修补接——禁绕径手建；
4. 观察面：舱面/榜单若该显示 xvenue paper 户而未显，同票补
   （可见面人话口径照 T-0371）。

## 验收锚

① fin_strategy 见 xvenue-carry 主从 paper 行（venue=okx+mt5 各
  一）+fin_paper_fill 开流；
② mt5 腿 swap 端记账对拍（信号快照值 vs 记账值）；
③ 链路断点清单落回填（有断=根修记录，零断=如实报零断）；
④ check_all 绿戳对口收口 SHA。

## 红线

- 不碰真单径（MT5 OrderSend 类 mutating 面零触）；
- 不动业主凭据/账户配置；
- 不动资金上限键（T-0376 域）。

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
