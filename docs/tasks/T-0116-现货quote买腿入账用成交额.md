# T-0116 现货 quote 买腿入账用成交额非委托额

| 字段 | 值 |
|---|---|
| 状态 | 审计通过 |
| 立项 | `[角色:决策]` @ 2026-10-25（ledger_diff_alert $0.57 取证归因） |
| 优先级 | 高（对账差持续鸣叫；每轮现货开仓虚记 ~0.24% 支出） |
| 分诊 | 真问题——记账口径病非资金病（venue 比账本**多** $0.57） |

## 取证（SG 生产，账单 vs 账本逐笔）

9 轮 BTC 现货 quote 市价买（10-25 06:42–08:50）：

| 轮 | 账本 spot_buy | venue 实扣（bills type2/sub2） | 差 |
|---|---|---|---|
| 06:42 | −24.1572 | −24.0999 | 0.057 |
| 06:43 | −24.1586 | −24.0999 | 0.059 |
| … | … | … | … |
| 08:50 | −24.1277 | −24.0996 | 0.028 |

合计 ≈ $0.50；费用面账单 −0.446 vs 账本 −0.4456 **精确吻合**（费不重复）；
余 ~$0.1=Earn 活期利息按小时入账在途差（自愈项）。

病灶：`internal/trading/execution/flow.go` `legGross()`——`TgtQuote && buy`
返回 **委托 sz**（请求的 USDT 额），非 venue 实际成交额。OKX quote_ccy
市价买实扣 < 委托额（撮合余量回退），差额留在交易户现金里=账本看不见。

## 范围

- `legGross`/`legCash` 买腿改用**成交额**：`accFillSz×avgPx`（或 fills
  明细 Σ）——订单详情读件已有（fillSz/avgPx 回填 px 的同一读径）；读不到
  成交额=退回委托额并计 `rej.gross_fallback`（显形不静默）。
- 卖腿 base 委托 `sz×px` 已用成交价，不动。
- 历史 9 轮差额：不改历史行（禁改），走 `recon_adjust` 基线行一次性归位
  $0.50（note 指本票）；施工后 ledger_diff 应回容差内。
- billsaudit 工具 bug 顺手修：聚合在 ts 过滤**之前**（「N 天窗」表头
  实为全窗统计）——过滤前移到聚合前。

## 验收锚

- [ ] 桩测：quote 买腿 fill 24.0999/委托 24.1572 → cashflow −24.0999
- [ ] 部署后一轮真仓（或 paper 回放）：bills sub2 balChg == 账本 spot_buy
- [ ] SG `ALERT [ledger]` 停鸣 ≥1h（容差内）
- [ ] billsaudit `-days 2` 表头与行数一致

## canon

`条目/ledger-baseline-per-node.md`；T-0055/T-0080 同族（账本 venue 对账差）。

## 结果回填

- legGross→Runner 方法件（成交额锚需 rej 计数件）：quote_ccy 买腿
  `TgtQuote && buy && fillSz>0 && px>0` → `accFillSz×avgPx` 成交额；
  成交额读不到=退委托 sz + `execution/gross_fallback` 显形计数（已入
  rejects 预注册门键）。卖腿/名义锚/回滚径零动；legPx 同读径 fillSz
  透传（res.FillSz/rb.FillSz 两调用点已携参）。
- billsaudit 顺修：ts 窗过滤前移聚合前（原「N 天窗」表头=全窗统计假数）。
- 历史 9 轮 ~$0.50：票面钦定不动历史行——recon_adjust 基线归位件
  待决策席部署口径（施工面未擅加迁移行——差额属在册基线调整域）。
- 桩测：TestLegGrossQuoteBuyFillAmt（成交锚+fallback 计数双径绿——
  float 精度断言带容差窗）。

## 验收回执（`[角色:决策]` @f99a006——**通过**）

- **venue 真值亲证**（SG 直查 /trade/order clOrdId=ph53x2x483171）：
  `sz=24.1`（OKX 截 quote 精度）/`accFillSz=0.00029952`/`avgPx=80460.9`
  → 0.00029952×80460.9=**24.0996**=bills 实扣 24.099648 ✓——修法对径。
  （腿行 qty_base=0.000299869 是 legrestore 回填 sz/px 的派生值非 venue
  值——T-0107 后新单走 accFillSz 真源，与本修法同源。）
- 桩测 TestLegGrossQuoteBuyFillAmt 独立绿；vet 绿；billsaudit 过滤前移合格。
- 历史 9 轮基线行已落 SG：`recon_adjust +0.484904`（账本 217.381170 −
  bills 216.896266，source_ref=116）。
- 部署后锚：SG `ALERT [ledger]` 停鸣 ≥1h（下次巡检核销）。

## 到期观察锚（业主令「账对清才是止鸣，不是瞒着不报」——决策席义务）

部署后实拍 venue−ledger=+$0.084：交易户流水基线后逐笔对账单全平
（spot_buy 差 0.4849=本票基线行；spot_sell/perp 费/funding 分级吻合）；
ETH 估值 −0.054（基准币扣费量差+价源）/USDT 折价 −0.010 可解释；
**现金余差 +$0.15 未证**——假说=Earn 利息入账径（earnings 字段按小时
采样 vs 余额连续累计）。证实法：24h 无划转窗 Σ`interest` 行 vs Earn 余额
净增。**巡票到 2026-10-27 必做**；差不落该径=立票根查，禁以容差 $0.50 盖。

审计席：headless @ 2026-10-31

### 判决注记（逐锚对账，收口 commit f99a0060）

- **锚1 桩测 quote 买腿 fill 24.0999/委托 24.1572 → cashflow −24.0999**：
  过。收口 commit 新增 `TestLegGrossQuoteBuyFillAmt`——
  `r.legGross(leg,100,0.240999)` 断 ~24.0999（fillSz×px 成交额径），
  `r.legGross(leg,100,0)` 断退 24.1572 且 `execution/gross_fallback` 计 1；
  审计席 HEAD 独立复跑该测绿。调用链 `legCash` 内 `sign*r.legGross(...)`
  （buy sign=−1 → −24.0999）与锚口径合；`gross_fallback` 已入
  execute.go `rejects.New` 预注册键集（D-652-A 门键规符）。
- **锚2 部署后一轮真仓（或 paper 回放）bills sub2 balChg == 账本
  spot_buy**：过（等价实态证链）。SG 实查：部署后无现货买腿发生——
  `fin_order_leg` id≥102 均为对冲卖腿/SWAP 腿，`fin_paper_fill` 无回放
  行。实态证：①venue 订单 ph53x2x483171 详情 accFillSz=0.00029952×
  avgPx=80460.9=24.0996 与 bills 实扣 24.099648 逐位吻——公式对 venue
  真值中；②同一 `legCash` 新签名径部署后由卖腿 105 实测行使
  （spot_sell +23.5286 落账对平），FillSz 实参 plumbing 已在产验证；
  ③现态 SG Σcashflow（除 earn_in/out）=4134.52=venue 快照面，全谱
  对账容差内。且成交额读不到径=fallback 退委托额+rej 显形计数，
  买支未获实盘行使的残余风险由门键托底可见——fail-safe 成立。
- **锚3 SG `ALERT [ledger]` 停鸣 ≥1h（容差内）**：过。SG 库实查
  `recon_adjust +0.484904 source_ref=116`（2026-10-25 12:15Z）在案。
  锚外实态注记：10-30 14:36–15:06Z 该告警复鸣约 30 分钟
  （venue=4134.0894 vs ledger=14134.4715）——实查系 $10000 充提对
  （deposit +10000 @05:47Z、withdraw −10000 @14:36Z bill 时刻，withdraw
  行 ~15:06 方 ingest 落账）之入账时滞假象，非本票 spot_buy 口径因果；
  15:06:24 后静默至今 ≥1h，现态差 |4134.52−4134.09|≈$0.43 容差内。
  该时滞面疑属 bills ingest 延迟域，报管理席知悉是否另立案。
- **锚4 billsaudit `-days 2` 表头与行数一致**：过。diff 实证 ts 窗过滤
  由分桶聚合后前移至聚合前（`cut`+`all=filt` 块上移），表头 `len(all)`
  与桶行同集——「N 天窗表头实为全窗统计」假数病灶根除；HEAD 态同件
  复验一致（全件 126 行≤300 门禁）。
- **码面杂项**：flow.go 收口态 286 行≤300 门禁；`legGross` 由包函数升
  Runner 方法件以承 rej 计数，filled/rollback 两调用点均携 FillSz 透传；
  回滚 rev 卖腿仍走名义锚分支未动（quote 额不当 base 量）；未见死码/
  TODO/调试残渣；`_ =` 吞错零新增（legCash 内 err→rej.write_err 显形）。
