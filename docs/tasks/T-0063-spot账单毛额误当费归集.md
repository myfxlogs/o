# T-0063 SPOT 账单归集毛额误当费用——对账反向差 1.70

| 字段 | 值 |
|---|---|
| 状态 | 完成 |
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
