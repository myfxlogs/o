# T-0044 funding 实收记账件——OKX bills→fin_cashflow.funding_settle 写入面

| 状态 | 完成 |
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

## 结果回填

**实现摘要**（zcode-glm @ 2026-10-24）：
- 事件驱动主径（D-0008）：`okxpriv` account/balance_and_position 帧触发 `SetBillsHook`（30s 节流+异步无取消 ctx——WS 读循环禁被 REST 拖死）+拨号后 boot 一次补漏（断连窗口 bills 拉齐）；无周期轮询。
- `okxexec.Bills`：GET /account/bills（instType=SWAP，limit=100/页，after=billId 游标；subType 173 资金费支出/174 收入=官方 docs-v5 核证非猜测；缺 billId 行拒收）。
- `internal/ledger/funding`：Ingester 增量拉取（首页→遇已入库 billId 切口，≤5 页）；`fin_venue_bill` 源表（迁移 0027，billId UNIQUE 幂等+代理 PK 作 cashflow source_ref）；分类入账 funding_settle（balChg，空退 pnl）/fee（fee≠0）；杂币不入账但源行留档；position_ref 按 inst_id+时窗归仓（无归仓=NULL 不拦记账）。
- cmd 装配：`billsSrc` 适配 okxexec.Bill→funding.Bill（cmd/core/bills.go——账本域不反向依赖交易域）。
- 档案：okx-private-ws skill 补 bills 钩能力行；迁移撞号二犯入工程经验库待归纳。

**测试证据**：
- `TestIngestFundingAndFeeIdempotent`（174/173-pnl 兜底/fee/无费单只留源/无归仓 NULL+同 billId 重放零新记）PASS
- `TestIngestPaginationStopsAtKnown`（游标推进+重放零重）PASS
- `TestBillsFetch`/`TestBillsMissingBillIDRejected`（参数/解析/拒无名）PASS
- `TestBillsHookOnAccountFrame`（account 帧触发钩+30s 节流+无关频道不触发）PASS
- 全树 `go test ./... -count=1` 绿；`-race`（funding/okxpriv）绿；build/vet/gofmt/lines 门全过

**偏离说明**：迁移号 0026 撞 T-0041（master 前进），改 0027 并 rebase origin/master；真单 08:00Z 首结对拍（验收标准②）=决策席触发面。
