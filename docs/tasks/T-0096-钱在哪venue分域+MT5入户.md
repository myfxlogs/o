# T-0096 「钱在哪」venue 分域+MT5 入户（NAV 完整性根修）

| 字段 | 值 |
|---|---|
| 状态 | 施工中 |
| 领取 | zcode-c9596 @ 2026-10-25 |
| 涉及面 | （同窗口两票共用） |

> 病灶实证：MT5 户 ~$3247（fin_cashflow 开业注资 mt5/USD 3247.41 在账）——
> NAV 与卡面全无（NAV 2142 只计 OKX 三面）。业主令：venue 维度重构+未来新增所可复制。

## 钦定设计（venue→分区→行 三级）

```
OKX
  交易户   USDT $76.7 · ETH $23.6 · 遗留 N 种 ≈$X
  资金户   USDT $0.2
  Earn     USDT $2032.3
MT5
  户 3001157   $3247.4（净值；可用余额附标可选）
<未来venue>——同构复制，零新增版式
```

- **数据面**：MoneyRow 增 `venue` 字段（okx/mt5/…）；section 降为二级
- **MT5 行**：每真实账号一行（login 枚举同 T-0088 径——AccountSummary 现读，多账号逐会话读，读败/无会话=该 venue 区显「读件缺」不藏）
- **NAV**：`nav_total += MT5 净值合计`——基金全资产口径归真
- **前端**：venue 头（logo 复用 T-0088 件）+分区+行三级；venue 块按某定序（okx→mt5→后续注册序）
- T-0095 托管口径照旧生效（Earn/资金户只认 USDT）——两票同改同一组文件，窗口共用

## 验收锚

- NAV ≈ 2136+3247 ≈ 5383（对拍两边同纳 MT5——ledger 侧 cashflow mt5/USD 已在账）
- 卡面 OKX/MT5 两 venue 块；MT5 户行 = 真净值现读
- MT5 读件缺席=venue 区显「读件缺」非假零


## 业主追改 v2（同 T-0095 v2 口径）

- 卡面三级不变（venue→分区→行），但**全图零遗留行零遗留概念**——三区只显 USDT+在持腿币行
- MT5 户全计（基金专属券商户=全部基金钱）不变
- NAV 预期 = 76.7(交易USDT)+23.6(ETH腿)+0.2(资金)+2032.3(Earn)+3247.4(MT5) ≈ 5380


## 结果（施工回填 zcode-c9596 @ 2026-10-25）

- 实现：proto `MoneyRow` +`venue`/`note`（wire 兼容，legacy 字段退役保留位）；
  卡面 venue→分区→行三级（okx→mt5→注册序，venue 头 logo 复用 T-0088 VSVG
  件）；MT5 行=`mt5LiveFor`（OpenSession 逐会话 AccountFacts 出 login +
  AccountSummary 出净值现读）；`nav_total += MT5 净值合计`——基金全资产口径
  归真；读件缺/读败=venue 区「读件缺」行不藏（禁假零）；main.go mountUI/
  mountCore mt5Live 注入。
- 测试：`TestOverviewFundScopeVenueCard`（MT5 户行=净值现读随 NAV）、
  `TestOverviewMissingAccountsZero`（nil/读败=读件缺行两形）。
- 偏离：①票面 v1 示例「遗留 N 种 ≈$X」与 v2 追改冲突——按 v2 执行零遗留；
  ②MT5 多账号=每真实账号一行（当前凭据=单账号，枚举面就绪）；③Overview
  每调建 MT5 会话（30s budget 死线——读败 WARN+读件缺不悬挂）。
