# T-0116 现货 quote 买腿入账用成交额非委托额

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
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
