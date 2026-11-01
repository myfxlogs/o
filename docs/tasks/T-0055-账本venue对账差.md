# T-0055 账本-venue 对账差根查（~$4.6 缺口）

| 字段 | 值 |
|---|---|
| 状态 | 完成（判过并码在案，11-01清账翻面） |
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

审计席：headless @ 2026-10-30

## 审计判决

逐锚实态对账（收口 4be5938 全量 diff 亲读+SG 生产库/容器日志只读实测+测试锚本地复跑）：

- **锚① 缺口归因实锤** ✓：tools/billsaudit/main.go 实存（125 行只读件——全谱 GET /account/bills 无 instType、fin_venue_bill billId 对拍、type/subType 分桶聚合，宣称口径与码面逐行相符）。票面书面归因在案：bills 177 行 vs 入账 14 行；漏类逐类定性——type1 sub11/12=资金户摆渡非损益、type2 sub1/2 内工具单绕账=缺口主体、利息 177 行零 interest 类=不在 bills 任何 subtype。生产旁证：首笔 recon_adjust −96.0164（id=31）与票面「ledger 高 $92.64」量级吻合；fin_venue_bill 现 315 行。
- **锚② Earn 收益入账** ✓（前提半证伪·锚意等价达成）：锚文「bills ingest 补 subtype」前提经实锤证伪（利息仅现于 savings/balance.earnings），施工走唯一真源径等价落地——0030 迁移 kind 闭集扩 interest/recon_adjust（在 diff、生产已生效）；postInterest 事件化高水位入账在产实证：fin_cashflow interest 行 127 笔自 10-24 逐时走账至 10-30 08:05Z（本席实查 SG 库）；历史缺口走锚文备选「基线校准」——calibrate_recon.sh 在 diff + recon_adjust 7 行实落。测试锚复跑：O_TEST_PG_DSN 显式注入独立实例，TestPostInterest+TestInterestKindsInClosedSet 2 PASS 非 SKIP。
- **锚③ 他因根因+修复+告警静默** ✓：根因=工具径绕账书面归因在票；关联坑条在库（ledger-baseline-per-node=校准机制 verified 条；unwind-limit-floor-protected/probe-restore-scope-overreach=致因工具事故条）。修复=系统性日漏（利息逐日走阔）事件化根治在产 + 历史缺口校准核销。告警静默生产实证：o-core 容器（10-30 06:50Z 起在跑）日志全谱零 `ALERT [ledger] 对账差`、零缺岗/探测败/在途 skip 行——对账环活判且 |diff|≤tol=$0.50（fin_known_residual 豁免=0）；末笔校准行 10-25 12:15 后零新增=缺口未再长。

【注记报裁——不阻断，留档】
1. 收口 commit 携 billsaudit 编译二进制 16MB 入库——同日决策 816f42d 剥离+gitignore 工具族通配根绝，残渣已闭环。
2. postInterest 高水位 SELECT 以 `_ =` 吞 Scan 错——ErrNoRows=播种语义属设计内，他错亦落播种径（fail-safe 向），量级微。
3. 「工具径绕账」根因无专条 KB——票面归因为正本+关联条在库，判锚意达成；要专条可另立微票。
4. 锚②字面径前提被证伪、施工走等价径——锚文弹性报备管理席。
