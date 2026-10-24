# T-0044 funding 实收记账件——OKX bills→fin_cashflow.funding_settle 写入面

| 状态 | 施工中 |
| 领取 | zcode-glm @ 2026-10-24（worktree /opt/phase-wt/t0044，支 t/0044-funding-bills） |

## 病灶
`funding_settle` 被复盘/报表读但无写入件——live 持仓的 funding 实收不落账=收益面瞎（首仓 08:00Z 首结将至）。

## 施工面
- 私有流 `balance_and_position` 或 REST `/api/v5/account/bills`（type=funding fee 族，instType=SWAP）为源——**事件驱动优先**（WS 到账事件推），REST 补漏只作兜底对账（D-0008：禁轮询主径）
- 幂等键：venue billId→fin_cashflow dedupe（重启重放不双记）；position_ref 关联（按 inst_id+时窗归仓）
- fee 行同理：开平仓 taker 费入 fee kind（venue bills type=fee 族）
- 测试锚：bill 事件→funding_settle 行幂等断言（同 billId 双发不双记）

## 验收锚
- [ ] 测试锚绿
- [ ] 08:00Z 首结后 fin_cashflow 见 funding_settle 行（金额=venue bills 对拍）
