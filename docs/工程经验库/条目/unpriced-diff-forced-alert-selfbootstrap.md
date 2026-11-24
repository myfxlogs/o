---
id: unpriced-diff-forced-alert-selfbootstrap
title: 无价键账实差强制式=豁免自举怪圈+隐含价像——币量豁免先扣/鲜价折算/无价如实报（O-38064 根修）
kind: pit
date: 2026-11-24
source: 工单O-38064（ledger_diff_okx_NFT 六阶恒等告警额=0.5+豁免+1 实盘取证）
---

# 无价币种的账实差不得强制造美元数——币量空间先豁免、鲜价折算、无价如实报

**坑形**（三合一，同一判差块病灶）：
1. **无价强制告警式自举**：非本位键读不到价时 `diff=tol+exempt+1`
   ——豁免加多少告警涨多少（NFT 六阶恒等实证：豁免 0→告警 1.5、
   50→51.5），USD 定额豁免对币量差结构性无效还反喂告警额。
2. **USD 定额豁免追币量差**：`fin_known_residual.amount_usd` 是
   美元口径，币量差（NFT/BTC）根本不在同一空间——拿 USD 数去压
   币量差=单位错配（且 qty 语义行落进「未注册 resolver→fallback
   加锚」会把 amount_usd 当 USD 豁免双计）。
3. **取价无鲜度界+币量冠 $**：旧查询无 ts 界吃到陈 7 日行
   （BTC-USDT 64356.9 @11-17 vs 现价 60369.8）；底稿 `账实差 $%.2f`
   把币量差印成美元数、`0.0217 vs 0.0200` 被两位小数抹平。

**修法钉死**（`internal/ledger/diff_px.go`+`residual_tracked.go`+`run.go`）：
- `resolver='qty:<units>'`=**币量豁免**——币量空间先扣不经价；
  `trackedExempt` 显式分支 Mode="qty"/Value=0（不入 USD 豁免额、
  成分段照列）；只在非 USDT/USD 键生效，本位键照旧 fallback。
- `diffPx` 两级鲜价源：`md_tick_scan`（venue 本键，ScanPoller 60s
  拍库内现成）→`md_ticks`∪`md_mark_index`，皆须 `ts>now()-10min`；
  查询 err 禁吞（`_ =` 判例）→rej recon_probe_failed+WARN。
- 判差三分流：`rq=dq−qtyExempt`；rq≈0→不告警不查价；有鲜价
  `diffUSD=rq*px` 对 tol+exempt(USD)；**无价 fail-closed 照报但
  不产出美元数**（删 tol+exempt+1 式——无价≠静默）。
- 文案币量+币名主量：qtyFmt≤8 位去尾零；有价附 ≈$x.xx、无价写
  「无可用行情价/折不出美元金额」；币量永不冠 $。

**判据**：USDT/USD 键判差与文案逐字节守恒（A6 回归锚）；
变异证明=摘鲜度界→陈价键红（A7 实证）。
