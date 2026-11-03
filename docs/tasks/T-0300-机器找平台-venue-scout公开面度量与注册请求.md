# T-0300 机器找平台（venue scout）——公开面零凭据度量全球 venue 可实现净 APR＋达阈自动发「请注册」

| 字段 | 值 |
|---|---|
| 状态 | **施工中**（devin-t0300 @ 2026-11-03） |
| 立项 | `[角色:管理]` devin-cli @ 2026-11-03（D-0043）；会签 `[角色:决策]` devin @ 2026-11-03 |
| 领取 | devin-t0300 @ 2026-11-03 |
| 优先级 | 高——骨架现立；放量（加 venue 源、加族）在第一分钱之后 |
| 涉及面 | 新包 `internal/research/scout/`、新表 `fin_venue_candidate`/`md_venue_funding`（md 面）、新 proto（榜单只读面）、`cmd/core` 挂臂、条件请求出口（现有业主通知径）、`skills/研究员/SKILL.md` 触发面（第一分钱后改） |
| 业主令原文 | 「我想要机器自己找平台/交易所，我再去注册拿凭证，这才是真正的走出去了」（2026-11-03） |

## 背景

D-0016 Ⅲ 把「走出去」交给 LLM 研究员读资料，结构上走不出去（无数字）。真正的走出去＝
**机器用公开行情度量世界上每个可及 venue 的肥度，告诉业主该去哪家开户**。全部主流
衍生品所的 funding/标记价/指数价/费率表都是**无凭据公开端点**——度量不需要先注册。
业主实证「单所肥度 OKX 最高」是某时点的人工比对；肥度随 regime 变，且**跨所 funding
价差**（两所反向永续、零方向敞口、无现货腿）这一族只有两所同在才存在——现无从测。
本票让机器持续量，业主只在达阈时动手注册。

## CTO 会签注记（2026-11-03）

- **口径钉**：三族净 APR 的费用项一律走 T-0297 同源函数（`funding.RoundTripBps`
  /`CostAPR`——禁 scout 包自造第三份算式）。跨所 funding 价差族双腿皆永续
  （无现货腿）：`cost_rt_bps=2×(taker_A+taker_B+slip_A+slip_B)+fail+window`——
  票面「≈单所 2/3」只是量级示意，实现按显式式子算。
- **推荐阈基线**：跨所价差族**没有「OKX 同族」可对照**（该族两所同在才存在）
  ——该族基线=Earn 底；单所 carry/基差两族基线=`max(Earn 底, OKX 同族 30d 中位
  top-N)`。判据行写明 baseline_source（earn|okx_family）进 evidence。
- **持仓期**：单所 carry 的摊销持仓期键缺省取 OKX `guard.max_hold_hours` 同值
  （跨所各 venue 无 guard，用同一地平线才可横向比）。
- **表/迁移**：`fin_venue_candidate` 归 fin 链（fin_0097 序取用）、`md_venue_funding`
  归 md 链（md_1080 序取用）；`scout.recommend_margin_apr`/`scout.persist_ratio`
  两键随 fin 迁移 seed；`fin_opp_template`/`fin_proposal` 零写（本票只量不动钱）。
- **proto scoutv1**：骨架票内榜单面允许 stub（读 fin_venue_candidate 直出 top-20），
  codegen 走既有 buf 径；前端表非本票锚。
- **禁线强调**：任何 venue 私有/需凭据端点调用=立即挂起票；`rg` 对账锚不变。

## 钦定目标

### A. 候选 venue 表（人读机读两用）
`fin_venue_candidate`：venue / kind（cex-perp/cex-option/dex-perp/cfd-broker）/ 公开行情端点
（funding/mark/index/instruments）/ maker_bps / taker_bps / 最小名义 / KYC 可及性（业主地区）
/ API 交易可及（有无官方 REST+WS）/ 提币限制 / evidence（费率页 URL＋抓取日期）/ state
（candidate/measuring/recommended/registered/rejected）。**首批 seed**（公开面均无凭据）：
Binance USDⓈ-M、Bybit、Bitget、Gate、KuCoin Futures、MEXC、Hyperliquid、dYdX v4、
Deribit（期权/永续）、OKX（基准行）。CFD 经纪商 swap 表归二期（多为登录后页面）。

### B. 公开面采集（md 面，受 `net.budget_kbps` 闸）
`md_venue_funding`：venue / inst / ts / funding_rate / next_funding / interval_h / mark / index。
每 venue 每结算周期拍一次全 USDT 永续（非逐 tick——度量用不着）；instruments 面日拍。
各 venue 公开端点 URL/字段方言落 `skills/venue-scout-<venue>/` 能力账（同 okx-market-ws 格式，
**协议面取证 skill 纪律**：下「无此面」结论前消息体全文核证）。

### C. 度量（确定性，零 LLM）
每日对每个 measuring venue 算三族、输出 30d 滚动：
1. **单所 carry**：top-N（N=5）净 APR＝|funding|×settles×365 − 费用年化（口径**同 T-0297**：
   往返费/min_hold_days×365）。
2. **跨所 funding 价差 vs OKX**：同币对两所 funding 之差×settles×365 − 两所永续腿费用年化
   （无现货腿，费用≈单所 carry 的 2/3）。
3. **基差**（有季货的 venue）：同 T-0298 口径。
榜单行＝venue × 族 × top-N 中位净 APR × 30d 持续天数（净 APR>Earn 底的天数占比）。

### D. 推荐阈与条件请求（自动，业主只收结论）
`scout.recommend_margin_apr`（seed=0.05）：某 venue 某族 **30d 中位 top-N 净 APR ≥ OKX 同族
＋margin，且持续占比 ≥ `scout.persist_ratio`（seed=0.6）** → state→recommended，向业主现有
通知径发一条**条件请求**：「请注册 <venue>：实测 30d <族> 净年化 X% vs OKX Y%；需 API key
（读＋交易，**不开提币**）；费率档 maker a/taker b；KYC 要求 …；注册后把凭据放 <能力账径>」。
同一 venue 一次请求后 30d 内不重发；业主回「不注册」→state=rejected（留因）。

### E. 研究员席唯一留职（第一分钱后接）
recommended 事件触发研究员短会话一次：读该 venue 公开文档，起草**注册清单**（费率页/API 文档
/KYC/地区限制/最小入金/已知坑）落 `fin_venue_candidate.evidence`——这是 LLM 该干的活（读文档
整理），不是找机会。

### F. 榜单只读面
proto `scoutv1`（unary 快照：venue 榜 top-20）挂现有 Connect 栈；业主前端一张表即可（P4 面
顺带，不另立前端票）。

## 验收标准

- [ ] `fin_venue_candidate` 首批 ≥8 行 seed，费率 evidence 全带 URL＋日期
- [ ] ≥3 个非 OKX venue 公开 funding 面实采 7d（md 行数/新鲜度实拍），带宽实测 ≤ `net.budget_kbps`
- [ ] 度量三族对 OKX 基准行自算结果与 fin_opp_eval 现有 funding 族 top-5 对齐（误差 <5%——同口径自校）
- [ ] 跨所价差族对至少一对 (OKX, X) 出 30d 榜单行
- [ ] 达阈→条件请求一次性发出实证（可用调低 margin 的测试键触发后恢复）；30d 不重发单测
- [ ] 零凭据：全票无任何 venue 私有端点调用（`rg` 对账 `/api/v5/account|/private` 零命中于 scout 包）
- [ ] 业主地区不可及 venue（KYC 列）不进 recommended

## 范围外/禁线

- **不注册、不下单、不放凭据**——注册是业主物理动作；凭据到手后加 venue 执行面＝另立票
- 不做 LLM 选所——推荐只由数字触发；LLM 只起草注册清单
- 不碰 OKX 现有采集/评估链（只借其 funding 族作基准对照）
- CFD 经纪商 swap 表、DeFi 借贷利率归二期（本票 seed 行可留 state=candidate 不量）

## 参考

D-0043 / D-0016 Ⅱ（文法×能力账——新 venue 登记能力账后自动出族）/ T-0122（宇宙放开实现样式）
/ `skills/okx-market-ws`（能力账格式范本）/ `skills/协议面取证` / T-0297（费用年化口径真源）

## 结果回填

施工 devin-t0300 @ 2026-11-03，分支 `t/0300-venue-scout`（本仓无 origin 远端，分支提交即锁据）。骨架票范围按 CTO 会签钉执行。

- **A 台账**：`fin_0098_venue_scout.sql`——fin_venue_candidate 全列（kind/endpoints/
  maker/taker/min_notional/kyc_ok/api_ok/withdraw_limit/metrics/evidence/state/
  last_request_at）+首批 10 行 seed（3 行 measuring：binance-usdm/bybit/hyperliquid；
  okx 基准行；余 candidate）+scout.{recommend_margin_apr=0.05,persist_ratio=0.6,
  top_n=5,measure_interval_h=24} seed。
- **B 采集**：`md_1081_venue_funding.sql`——md_venue_funding hypertable+90d 保留账
  （audit_keep）；`scout/collect.go` 三 adapter 公开端点（Binance premiumIndex/
  Bybit v5 linear tickers/Hyperliquid /info metaAndAssetCtxs）——全 USDT 永续一拍，
  零凭据零头；Fetcher 缝可桩。
- **C 度量**：`scout/metrics.go`——三族：单所 carry（|funding|×settles×365−
  funding.CostAPR 同源）；跨所价差 vs OKX（同符号 funding 差×settles×365−
  显式式 2×(takerA+takerB+slipA+slipB)+fail+window）；basis 族骨架期缺季货行情
  面=如实缺（board 行不生）。榜行写回 metrics jsonb（family/net_apr_mid/persist/
  baseline/baseline_source/top_n）。
- **D 推荐**：maybeRecommend——kyc_ok=false/registered/rejected/30d 重发闸全断；
  达阈→state=recommended+last_request_at+Alerter 条件请求（费率/KYC/不开提币口径）。
- **E/F**：研究员触发面属第一分钱后（范围外如实）；proto `scoutv1` Board stub
  直出 top-20（buf.gen 双模板已含 proto/scoutv1，gen 件在支），
  `scoutv1connect.RegisterScoutServiceHandler` 挂 mountCore。
- **挂臂**：`cmd/research/dcd.go` researchArms 挂 scout 臂（公开面零凭据）。
- **测试**：3 桩（桩 fetcher 三 venue 落行/达阈推荐+30d 不重发/KYC 禁）全绿@5434。
- **待外部位**：7d 实采行数+带宽实拍、跨所价差 30d 榜单行、OKX 同族 5% 自校、
  达阈实证——皆部署后实证件（venue 出向在开发机，本机不代产测）。

## 返修① 回填（devin-t0300，同支续提）

- **①canon 归一**：`canon()` 剥方言→base ccy（-SWAP/-PERP/分隔符/USDT-USDC-USD
  尾/1000PEPE 类数量前缀），join 键全走 canon；`TestCanonNormalize` 8 组含
  1000PEPE 双向锚。
- **②OKX 榜读正本**：`loadVenue` venue='okx' 分支读 `md_funding`（inst_id 列），
  非 scout 自采行。
- **③baseline 实算**：carry 族 `max(Earn(funding.EarnRate 共用读件), OKX 同族
  30d 日序列中位)`，BaseSrc=earn|okx_family 进榜；xvenue 族=Earn 底。
- **④persist 真序列**：`dailySeries`——每 (canon,day) 最新行→每 day top-N 中位
  日值；`NetAprMid`=30d 序列中位；`Persist`=日值>baseline 天数/30（缺测日按未
  达标计）。xvenue 同构（canon×day 内积 join）。
- **⑤双腿费读台账**：xvenue cost `2×(taker_A+taker_B+2×slip)+fail+window`——
  takerA/B 读 fin_venue_candidate.taker_bps（缺/≤0=config taker 回退不虚构）。
- 测试：scout 4 桩+funding 11 桩全绿@5434；funding.EarnRate 导出委托（与
  T-0298 支同形，合并无歧义）。
