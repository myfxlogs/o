# T-0096 「钱在哪」venue 分域+MT5 入户（NAV 完整性根修）

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
