# T-0012 生命周期状态机与 paper 沙箱（M1 链 G3→G5）

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 提出 | `[角色:决策]` 2026-10-23 |
| 类别 | 投资域/研究域衔接（M1 链） |
| 依赖 | T-0010✅（fin_proposal 提案面）；`策略评估与仓位` 技能必查 |
| 领取 | |

## 背景

M1=OKX funding 对冲打穿 G0→G11。G0→G2 已通（T-0007/T-0010）；本票竖切 **G3 paper 沙箱→G4 晋升→G5 配分** 的生命周期最小机。funding 提案过阈后不能直通真单——paper 段是「全流程对真市场数据空跑」的验证带，判据机读、晋升成记录。

## 范围

1. **`fin_strategy` 表**：状态机行——PK、inst_id、template_ref、proposal_ref、state（闭集 CHECK：`proposed/paper/probe/live/degraded/retired`——按 `架构约束.md` 生命周期态；`enumerated` 态若用亦入闭集）、entered_at、promotion_criteria 快照（jsonb）、状态迁移日志（同表 audit 列或 `fin_strategy_transition` 子表）。
2. **状态机本身**：迁移合法性枚举校验（非法迁移拒绝+计数；禁止回跳、禁止越过 paper 直升 live——G4 第一笔真钱须投资委员会开关位 `app_config` 键，缺键 fail-closed）。
3. **paper 沙箱执行体**：paper 态策略对真实行情流（md_funding/md_ticks 事实行）做空跑记账——虚拟开平仓+虚拟费用五拆，产出 `fin_paper_fill`（或 eval 扩展面）记录「假如今晚会怎样」。**禁碰真单径、禁引 execution 包**——import 方向闸把关。
4. **晋升判据**：paper 满判据（时长/模拟净回报/回撤上界——键全走 `app_config` `invest.lifecycle.*` 热启，缺任一键=不晋升）→ 状态翻转+晋升记录；probe/live 态行生成但真单仍禁（G6 风控闸下一票）。
5. **员工**：`命官`（投资域生命周期官）Register+BeatLoop(30s)，产出窗=fin_strategy 行/迁移记录。

## 验收标准

- [ ] `fin_strategy`+迁移日志表序进入库；state 闭集 CHECK；非法迁移被拒有测试锚（含 paper→live 越级拒）
- [ ] paper 沙箱吃真实事实行产出虚拟结算记录；**import 闸实证 research/invest→execution 0 边**
- [ ] 晋升判据全走 `app_config` 热启；缺键 fail-closed（缺键不晋升有测试锚）；晋升留痕可查
- [ ] `命官` 在岗（registry 行+心跳+产出窗）
- [ ] 机检全绿；CI 绿

## 范围外

- 风控闸实现（G6 下一票）、真单执行（G7）、账本入账（G9 起）；MT5 面；其他五模板生命周期（funding 型先通）
