---
id: residual-static-exemption-drift-tracked-resolver
title: 构性残差静态豁免恒滞后——活值残差须 resolver 现算非定额续命（O-36327 根修）
kind: pit
date: 2026-11-24
source: 工单O-36327/O-35882（#36307 差$0.98告警→#36312 定性→#20-22 人工+$1续豁三连实证）
---

# 静态豁免行对活值残差=永续滞后——豁免额须存解析键现算

**坑形**：`fin_known_residual` 静态 `amount_usd` 豁免对**活值残差**
（随市价/应计漂移的已定性差）恒滞后——残差每爬过豁口告警一次，
人工 `+$1` 续豁，豁免链滚动续命（#18→#22 五代实证）。静态定额
把「已定性」误当「已量化」——定性=机制归因，量化=时变量，
两者不可互替。

**机制分层**（修法钉死）：
- `amount_usd` 列降级为**锚值**——不再是豁免额本体，是
  |跟踪−锚| 破带判据的参照（锚=立案时实测快照）。
- 新列 `resolver` 存**解析键**（`dcd_mtm_drift:<ordId>`
  /`earn_accrual_gap`），豁免额=无 resolver 静态行 Σ + Σ resolver
  行**现算值**（`internal/ledger/residual_tracked.go` 注册表，
  resolver 经 Runner 注入读件——ledger 不识 venue 方言）。
- 现算源全部走既有账本/行情面：DCD 口径差=|venue 单名义量−
  `dcd_baseline` 凭据币量|×`md_ticks∪md_mark_index` 现价
  （venue 侧走 `WithDcdOrders` 注入读件=okcli.DcdOrders 60s
  缓存在案同源；名义在持三态 live/pending_settle/pending_redeem
  才计，settled/redeemed 走回款径不计；币种不配/单缺席/非在持态
  =err 回退锚）；
  Earn 应计差=`fin_cashflow kind='interest'` 累计和
  （earn-accrual-vs-venue-read-residual 条目机制定案——
  venue `amt` 面只认实现额，未实现应计=恒差增生源）。

**失败语义三分**（钦定 fail-closed）：
1. resolver 未注册/源读败 → **回退静态锚值**（未知不放大不缩小，
   同 run.go 豁免读败让位判例）；
2. |跟踪−锚| > max(2×锚, $25) → **破带不按构性续**：该行豁免
   贡献归零 + `esc_class='对账'` 升件（24h 同窗去重，
   scope=`residual_model_breach_<venue>_<ccy>_<key>`）——
   **检测力不稀释**：漂移超模型=可能有新成分，按新族异常处置；
3. 行集读败 → err 上抛，recon_probe_failed+本键让位（原语义不变）。

**已知盲区诚实标注**：earn_accrual_gap 现值=累计应计上界——
利息实现入 amt（复利/派发/赎回息差）无现金流事件可读，
实现后现值虚高→破带护栏兜（超带即升件转人工定性，
不静默吞）。dcd_mtm_drift 现值=名义量差×mark——
venue 产品 USDT-MTM 真值（期权型敞口非线性）归 OCI/唤知面，
本解析器量名义量差域（应计增厚=口径差增生主源已定案），
模型近似诚实标注同上兜。

**判别推广**：凡「已定性残差」豁免设计——先问残差是否活值
（随输入漂移）。活值→豁免行必须存 resolver 解析键现算，
静态定额=把告警频率外包给人工续命；死值（一次性定案差）
→静态行+expires_at 复报闸即可。混用判据：豁免链代次
（同 venue/ccy 同 reason 族续命 ≥3 代）=活值漏判实证。

**验收锚**（residual_tracked_test.go 六案）：BTC 价漂→跟踪值
随动非锚；interest 累计→跟踪值=应计和；破带→归零+升件落账；
未注册/源缺/读件缺席/单缺席/非在持态→回退锚+静态行照旧求和；
reconcile 级 mark 漂移→静默/照报转换实证。
