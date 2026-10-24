# T-0055 账本-venue 对账差根查（~$4.6 缺口）

| 字段 | 值 |
|---|---|
| 状态 | 完成 |
| 立项 | `[角色:决策]` @ 2026-10-26（T-0050 移交项③：recon 告警每帧洪刷 venue 2133.6 vs ledger 2129.0） |
| 领取 | zcode-glm @ 2026-10-26（worktree /opt/phase-wt/t0055，支 t/0055-recon-gap） |
| 优先级 | 中 |
| 涉及面 | `internal/ledger/`（recon 面）、`fin_cashflow` 行源、Earn 收益入账径 |
| 范围外 | recon 告警节流（洪刷是症状——先根查缺口） |

## 背景

SG 生产 recon 对账持续告警：venue 权益 2133.6 vs 账本 2129.0，缺口 ≈$4.6。施工席疑 **Earn 收益未入账**（Earn 利息按日结——收益走 bills subtype 进账，若 ingest 未含该 subtype=账本系统性偏低）。

## 验收标准

- 缺口归因实锤（Earn 利息/费用/划转漏类逐类实读 bills vs cashflow 对拍）
- 若=Earn 收益漏入账：bills ingest 补该 subtype 类+历史缺口一次性补记或基线校准
- 若=他因：根因条目化+修复；告警静默=缺口 <$0.5 或入告警白名单（有书面归因才算静默成立）

## 结果回填

**归因实锤**（zcode-glm @ 2026-10-26，tools/billsaudit 全谱对拍+分账解剖）：
- **bills 全谱 90 天窗 177 行 vs fin_venue_bill 已入账 14 行**：未入账类=type1 sub11/12（资金户划转——本金摆渡非损益，cashflow 口径本就不记 ✓）+type2 sub1/2（成交行——生产单损益已由 fin_order_leg 径覆盖 ✓；**工具单（okxunwind/探针 -mut）动钱绕过 fin_cashflow=缺口主体**）。
- 施工时点分账：ledger(okx/USDT)=2229.42（deposit 基线 2003.59+earn_in 301.5−earn_out 100.5−fee 0.14+trade_pnl 24.97）vs venue 现读 2136.78（trading 79.20+funding 0.20+savings amt 2057.37）——**ledger 高 $92.64**，与工具径损耗量级吻合（PENGU 事故往返 ~$12+T-0048 复审 -mut 台账损耗+unwind 价差费+申购赎回差额，正负相抵）。原 $4.6 告警=同源缺口的早期时点切片。
- **Earn 利息假设证伪一半**：利息不在 bills 任何 subtype（177 行零 interest 类实证）——只现于 /finance/savings/balance.earnings 字段，bills ingest 补类此路不通；利息=独立入账径缺口（金额小 ~$0.2/日 但系统性逐日走阔）。

**修复**：
- ①迁移 0030：kind 闭集扩 `interest`/`recon_adjust`。
- ②利息事件化入账（`ledger.Runner.WithEarnEarnings`+`postInterest`）：earnings 累计值 vs earn_interest 事件行高水位，delta≥0.005 落账（source=事件行 ts 毫秒——md_account_events 无代理 PK 实勘）；bootstrap 播种零入账。reconcile 对拍前先入账。
- ③基线校准径 `scripts/calibrate_recon.sh`：部署 0030+播种后一次性落 recon_adjust 行（amount=现场 venue−ledger）——历史工具径缺口书面归因后真值校准。
- cmd 接线：venue_readers 增 savings earnings 读件（WithEarnEarnings）。

**测试锚**：TestPostInterest（bootstrap 零入账/应计不足半分不落/delta 落账 source 断言）+TestInterestKindsInClosedSet PASS；全树 build/vet/test/lines 绿。

**校准执行（部署后必做——移交决策席随 T-0054 部署）**：CAL_DRY=1 先看数→落行→recon 静默判据 |差|<容差。
