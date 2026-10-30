# T-0063 SPOT 账单归集毛额误当费用——对账反向差 1.70

| 字段 | 值 |
|---|---|
| 状态 | 审计通过 |
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
- [x] 重放 12:41–12:47 绞肉时段 SPOT bills→cashflow：fee 入账总额≈bills fee 列总额（±0.01）。
- [x] bal_chg 毛额不再产生 fee 行。
- [x] 修后实盘观察 ≥2h 对账静默（live_switch 已开，自然有新腿产生）。

审计席：headless @ 2026-10-31

## 判决注记（逐锚对账，收口 commit=62417989+回填 7aa8964a，生产库只读复证）

- **锚1 PASS**：12:41–12:47 窗 SPOT USDT 侧 bills fee 列合计 -0.17481422336（7 行）＝同窗 SPOT 源 cashflow fee 行合计逐分相等；扩至全史 97 条 fee 行 `cf.amount=bill.fee` 逐行相等（mismatched=0）。±0.01 容差远满足。
- **锚2 PASS**：码面 default 分派仅 `b.Fee` 入 fee kind（T-0062 部署版即此语义，master 现行同）；数据面零 fee 行量级等于 bal_chg（±25 换形从未入账），ETH 侧费（ccy≠USDT）被 ccy 闭集正确滤除。
- **锚3 PASS**：修后期（≥10-24 13:15Z）30 条 fee 行合计 -0.480879182646 与 bills fee 列精确一致；账单续流至 10-25 17:00Z，10-25 07:35Z 账本重定基（recon_adjust 102/103）后 known_residual 仅剩 ETH 尘 ~$6（T-0080 归口在案），对账静默远超 2h。
- **根因更正（取证驳回票设）**：事故读数系观测口径错位——「venue SPOT bills 实净」混 bal_chg 双腿与 fee-only cashflow 异口径对比，叠加 SWAP/SPOT 混域；非码面把毛额当费。收口实质=注释钉死口径+回归测试 TestSpotFeeNotGross（双腿账单锚），属文档级防线非行为修复。
- **注记**：收口支 t/0063-spot-fee 未并入 master（注释+测试仅存在于支上），并入归 sweep_lander 串行收口；语义本身 master 已在位，锚不受影响。recon_adjust id=92 已被 10-25 基线件吸收。
