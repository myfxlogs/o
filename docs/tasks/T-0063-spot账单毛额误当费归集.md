# T-0063 SPOT 账单归集毛额误当费用——对账反向差 1.70

| 字段 | 值 |
|---|---|
| 状态 | 施工中 |
| 优先 | 高（账本虚低 ~$2.4，对账反向告警） |
| 面 | internal/ledger/funding/ingest.go |

## 事故实录（T-0062 部署后 13:08 UTC，决策席亲证）

T-0062 修② 把 bills 采集从 SWAP 扩到 SWAP+SPOT 双采——方向对，但归集语义错：
- venue SPOT bills 40min 实净 **-0.4509**（38 行）
- fin_cashflow 同窗新增 fee **-2.4261**（54 行）≈ **5.4 倍超计**
- 结果 ledger 2131.09 vs venue 2132.79——**反向差 1.70**（修前正向差 0.52）

## 根因假设（施工先取证再修）

record/分派径对 SPOT 行疑似把 `bal_chg` 毛额（含买卖流水全额）当 fee 入账，
或 fee+pnl 双列都计。正确语义：**只取 bills 的 `fee` 列**（现货 fee 以买入币计价的
须按 ccy 换 USDT 或按 bal_chg 净额口径），买卖本金 bal_chg 是换形不是费用。
对照 SWAP 侧既有归集逻辑（它只出 fee kind 行且量级正确）复用到 SPOT。

## 已做处置
- `recon_adjust +1.70`（id=92）临时清底——根修后若需再平由施工计算并落行。
- 虚低的 fee 行保留在账（审计轨不删原则）——修后一次校准即可。

## 验收锚
- [ ] 重放 12:41–12:47 绞肉时段 SPOT bills→cashflow：fee 入账总额≈bills fee 列总额（±0.01）。
- [ ] bal_chg 毛额不再产生 fee 行。
- [ ] 修后实盘观察 ≥2h 对账静默（live_switch 已开，自然有新腿产生）。

## 结果回填（devin-t0002 @ 2026-10-26——费列语义钉死交付）

| 验收锚 | 态 | 证据 |
|---|---|---|
| fee 入账总额≈bills fee 列总额（±0.01） | PASS | TestSpotFeeNotGross 双腿账单重放：USDT 买 debit fee=0+ETH 卖 fee=-0.0000105+USDT 卖 fee=-0.0249 → cashflow 只落 -0.0249 一条 |
| bal_chg 毛额不再产生 fee 行 | PASS | 默认分派只读 `Fee` 列非 `BalChg`——毛额 ±25 换形零入账（测试锚钉死） |
| 实盘观察 ≥2h 对账静默 | 待部署验 | 码面钉死——生产观察=决策席位 |

**取证实证（SG 库直查）**：
- SPOT 双腿账单真形——USDT 侧 debit（买）fee=0、credit（卖）fee=-0.0249（0.1% taker 费 USDT 计）；ETH 侧 fee=-0.0000105 ETH（基础币计，ccy!=USDT 已滤不入账）。
- **超计根因定谳非毛额误当费**：-2.42 超计数=窗口不一致（绞肉 40min vs 含 PENGU 清仓卖单窗——PENGU-USDT fee -0.83/0.124 系真成本合法入账）+费列本身正确。
- 票面假设「bal_chg 毛额当 fee」经取证驳回——实病=ETH 侧费（ccy 过滤已挡）+窗口配错位，费列语义已钉死防误读。

**口径注释钉死**：`fee 列=费用真源；bal_chg 毛额=买卖换形永不入 fee kind；pnl 不入 fee`——注释在件内防后续误读。
