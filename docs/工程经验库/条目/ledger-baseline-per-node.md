---
schema_version: 2
id: ledger-baseline-per-node
kind: pitfall
risk: medium
status: verified
keywords: [对账差, ledger_diff_alert, phantom alert, recon_adjust, 基线, 分账, 双端部署, venue equity]
scope: [internal/ledger, fin_cashflow, 部署拓扑, 本机+SG 双端]
summary: 分账部署下每节点 fin_cashflow 独立——新节点/新库无 venue 基线行时对账差=全部权益（phantom alert 连刷），立账=recon_adjust 基线行校准非修码
first_seen: 2026-10-25
last_verified: 2026-10-25
skills: [生产取证]
evidence_refs:
  - type: server_fact
    ref: 本机 o-core「ALERT [ledger] 对账差 okx/USDT venue=2135.58 ledger=-2.8154」每 12s 自 02:34Z 连刷 5h+
  - type: server_fact
    ref: 本机 fin_cashflow okx/USDT 净额=-2.8154（deposit/recon_adjust/spot_* 全缺，min ts=2026-10-24 04:07）
  - type: server_fact
    ref: SG fin_cashflow okx/USDT 净额=2108.73（deposit 2003.59+recon_adjust 链+双腿行在账）对账静默
---

# 分账部署 venue 基线各行其立——缺基线=phantom 对账差

## 病灶

双端部署（本机+SG）各自 postgres 分账：venue 侧对拍的是**同一真实账户**权益，
ledger 侧对拍的是**本节点自己的 fin_cashflow**。新节点/新库若无历史入金与校准行，
净额≈0（或仅残 fee/interest 负值）→ 对账差=venue 全部权益，介入级告警连刷。

本机实证：本地库 2026-10-24 04:07 起账，okx/USDT 仅 fee×92+earn_out+interest+funding_settle
（净 −2.8154），venue 实测 2135.58——假差 $2138 自容器起连刷 5h+。

## 判据

- phantom vs 真差分型：ledger 净额≈0 或仅小额负尾 + 库龄远小于账龄 = 基线缺席，
  **非 venue 钱丢**。先查 `SELECT kind,SUM(amount) FROM fin_cashflow WHERE venue=? GROUP BY kind`
  看 deposit/recon_adjust 有无。
- 修码面无用——这是数据面基线缺席，修=立账非改码。

## 处置（本席亲证有效）

1. 尾差归零：recon_adjust +|小尾差|（fee 净额等，source_table 标 baseline 系）
2. 本金基线：recon_adjust +现 venue 值（写时取最新 ALERT 行 venue= 现值，
   source_table=`baseline_local_o_*`）→ 下一拍静默
3. 残尘键（如残币归口半拍的 venue>parked 微差）走 fin_known_residual 豁免
   （USD 额+expires），勿写假 parked 行——台账行=真划转凭据不可伪造

## 防再发

- 新节点/新库上线清单加「对账基线立账」步——deploy SOP 面（T-0104 双端审计随行）
- 分账节点间 ledger 不可互搬：各自立账，净额应收敛同 venue 值但路径各自（本机
  无仓位无 mtm 件，开仓位后 mtm 漂移>tol 会再警=真信号）
