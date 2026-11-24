---
id: clordid-discriminate-all-selfaccount-paths
title: clOrdId 判别键须枚举全部自记账径——单查 fin_order_leg=双计案
kind: pit
date: 2026-11-25
source: 工单O-39310 施工（USDT 计价腿守恒破口——ec 清仓单双计实证）
---

# 判别键只查一条自记账径=漏判双计（O-39310 实证）

**坑形**：venue 账单计价币腿要判别「这笔 venue 侧活动 O 是否已另账
记过」——第一版判别只查 `fin_order_leg.cl_ord_id`+legCash 现金行
（execution 径），回填 361 腿全绿过闸。复核时发现 **residual_clear
（earn 杂币清仓）是第二条自记账径**：它不走 fin_order_leg，直接
`ledger.Post(spot_sell, source_table='fin_risk_decision')` 按
`intent->>'cl_ord_id'` 归因。18 张 ec* 账单被当 venue 侧补记
spot_sell+fee——与 clear 径已记 gross **逐分位相等**（TRX $9999.0813193
对 $9999.0813193）=纯双计 $10,025.89。

**判别**：凡「某键命中=已记过」型判别，命中集=**全部**会对同一外部
事件自记账的径，不是「最熟的那条」。枚举法=全仓搜该外部事件的
cashflow 写者（`INSERT INTO fin_cashflow`/`ledger.Post` 全量 grep），
按 source_table 分族——本案两族：`fin_order_leg`（执行腿）∨
`fin_risk_decision`（决策凭据腿）。

**修法**：判别谓词=两族并集（gross 域）；**分量级判别**——gross
（spot_*/unmapped_bill）与 fee 是两个账域：clear 径只记 gross 不记
fee、legCash 仅 feeCcy=USDT 记 fee——「gross 已覆盖」≠「fee 已覆盖」，
fee 域须独立查「同 clOrdId 有无 fee 现金行」，无则 bill fee 腿照记
（它是唯一 USDT 费账源，漏记=费黑洞）。回填侧同律+分量级幂等
（同 bill 已有任一 gross 域行即跳）。

**清理实证**：双计行=`DELETE` 本批误插 gross 行 18 条（fee 行保留
——fee 域本就该记）；跨表对拍矩阵 `(leg_cov,risk_cov,has_gross)`
四象限全落位=零覆盖+有账/有覆盖+无账，无「有覆盖+有账」交叉格。

**推广**：①判别键设计先列「谁会为这个事件记账」全集再写谓词；
②回填工具与 ingest 同判别口径（billprobe 镜像 spotQuote 双域）；
③append-only 下误插行的正解=识别后整行 DELETE 本批错误行（非
recon_adjust 冲正——recon_adjust 会进 deposit/withdraw 本金口径
污染已入分母），fee 等正确分量保留。
