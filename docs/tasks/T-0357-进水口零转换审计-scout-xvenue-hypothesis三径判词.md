# T-0357 进水口零转换审计——scout/xvenue/hypothesis 三径判词

- 状态：待验
- 领取：devin-t0357 @ 2026-11-05（业主令外派——审计票只读+判词）
- 级别：P3 审计票——进水口三条机会面扫描径全零出件，未判
  「市场无阈上件=正常空转」还是「断线=真静默病」前，
  产线产能统计无可信度

## 病灶（实查在案，勿再诊）

钱径审计（2026-11-05 第 1 性逐项）实测三进水口零转换：

| 径 | 实测 |
|---|---|
| scout venue 注册清单 | 9-venue funding 宇宙照常在采，venue 达阈
  注册请求零出件 |
| xvenue_carry → 提案 | md_signal 信号族在行，fin_proposal 面
  零提案化 |
| hypothesis 卡册 → 提案 | 假想官卡册在产，提案化转换零 |

三径同属「扫在转、单不出」——若系阈判据内正常空转则记账销案；
若系转换件断线=静默病（T-0343 同类病灶——该报的不报靠人发现）。

## 任务（审计非施工——本票交付=判词+必要时立修票）

1. 逐径 trace：信号产出→阈判据→提案写径全链代码走读；
2. 逐径取证：生产库信号行计数/阈值对拍/提案行存在性——
   判「数据喂到了但阈值不达标」vs「数据断供」vs「转换件死」；
3. 逐径判词落票面回填：正常空转（市场给不出阈上件）/
   断线（给出病灶点位+修法建议→另立修票）；
4. 若发现断线病灶：病灶点位+最小修法写进回填，
   **不擅自动手修**——修票另立走常径。

## 验收锚

1. 三径各一条「最近信号行存在性+阈对拍」实证行落回填
   （查询输出锚禁口报）；
2. 三径判词各一句定性（正常空转/断线-点位）落票面；
3. 断线径的修票建议节落回填（无断线=明示三径皆正常空转）；
4. check_all 绿戳对口提交 SHA（docs 面变更按门禁跑）。

## 红线

- 本票=审计票：不改任何运行时代码/配置/阈——只读+判词+建议；
- venue 全零 mutating；凭据读面走既有 READCRED 径禁新面；
- 判词必须有数据锚——「应该没事」类口报=驳回。

## 结果回填（devin-t0357 @ 2026-11-05）

### 审计方法

三径全链代码走读（scout/metrics.go maybeRecommend、sigdist/xvenue.go
Xvenue、hypothesis/{run,signals,xcarry,stakecarry}.go、execfam.go）
+ 生产库只读取证（SG o-postgres `docker exec o-postgres psql -U o -d o`，
全部查询输出锚如下）。研究臂宿主=开发机算力面（o-research.service，
T-0165/D-0029）——md_* 写本机 MD PG 经桥投 SG，fin_* 经 5544 隧道；
md_venue_funding 属 MD 域不在 SG o-postgres（已核实表缺席），其度量
产物=fin_venue_candidate.metrics 即阈判据终值，取证以 metrics 为准。

### 径① scout 注册清单——判词：**正常空转（阈未达，persist 全线不达标）**

链：collectAll→md_venue_funding→boardFor 三族榜→maybeRecommend
（NetAprMid≥Baseline+0.05 且 Persist≥0.6 双条件，30d 重发闸+kyc_ok）。

证据锚（SG o-postgres 实查）：
- `fin_venue_candidate`：11 venue 中 10 行 state='measuring'，
  metrics->>'measured_at'=2026-11-05T04:15Z（24h 日拍新鲜——臂活、
  measure 写径通）；ibkr=state'candidate' 无 metrics（未入 measuring）。
- 阈对拍（metrics->boards 逐行）：carry 族 net_mid 534%~1660% vs
  baseline 247.91%(okx_family)——净额条件多数达成；**xvenue_spread 族
  net 165%~282% vs baseline 6.22%(earn) 同达成**；但 **persist 全线
  ≤13%（阈=60%）**——binance/bitget/bybit/gate/kucoin/mexc carry 与
  xvenue_spread persist=10%，dydx=7%、okx=13%、hyperliquid 空——
  30d 序列中超基线日占比远不达 0.6，maybeRecommend 双条件恒假→
  state 合法滞留 'measuring'、last_request_at 全 NULL=零请求系判据
  内空转非断线。deribit/dydx/hyperliquid persist 空=序列不足如实缺席。

### 径② xvenue_carry→提案——判词：**正常空转（钦定血统闸稀疏供给+双层设计内拒）**

链：sigdist.Xvenue→md_signal{carry_diff,xvenue_spread}（bridge.xvenue.map
3 对在册）→hypothesis.xvenueCarryScan（Fresh 120s 闸）→|net|≥300bps→
卡+eval→proposed→fin_proposal（execfam.Has('xvenue-carry') 前置拒兜底）。

证据锚（SG o-postgres 实查）：
- 信号面活：`md_signal` carry_diff 2 key（ETH 12.1/BTC 31.7bps）、
  xvenue_spread 3 行 0.0h、funding_view 525 行 0.0h、depth5 11 行 0.0h、
  mt5_swap 3 行——sigdist 臂在产。
- **carry_diff 新鲜窗结构性稀疏（钦定）**：行 ts=least(上游)——
  mt5_swap 为 4h 拍面，其 ts 传染使 carry_diff 恒>120s 陈，
  xvenue.go:131-132 注释明载「mt5_swap 陈=carry_diff 陈，派单钦定」；
  最新 carry_diff=2026-11-05 04:47Z（≈1.9h 前）→Fresh 闸视缺席=
  扫描面多数时间窗内零信号，非断供。
- 转换件历史实证在案：`fin_opp_eval template='xvenue-carry'` 3 行
  （10-26/29/30）verdict=rejected、reason∈{net_ann_economics,
  mt5_leg_capability_missing}——BTC 净 +433bps≥300 曾过阈起评，
  拒因=mt5 trade-order 能力面未启（fin_capability_surface enabled=0）
  与经济学负值，均为真判据非死件；eval→proposal 写径代码在案
  （xcarry.go:226-237）。
- 兜底闸：`execfam.Has('xvenue-carry')=false`（execfam.go:15——
  T-0145 执行径未派案钦定不在册）→即便 proposed 也前置拒
  exec_family_missing 不入提案队列（T-0330 设计）。
- fin_opp_hypothesis xvenue_carry 卡 1 张 state='parked'（10-29）。

### 径③ hypothesis 卡册→提案——判词：**正常空转（批准径人工闸队列态）**

链：gap 驱动 scan（fin_surface_gap pending→卡）+信号臂
ScanSignals（funding 背离/depth5/xvenueCarry/stakeCarry→卡+直评）
→卡 state='proposed' 待决策席审→adopted→模板注册+enabled→
评估臂→fin_opp_eval proposed→fin_proposal。

证据锚（SG o-postgres 实查）：
- 卡册在产：`fin_opp_hypothesis` 13 行——proposed 1（cmc/
  stake_carry_template，11-04 最新）、adopted 1（okx/savings-rate，
  11-02）、parked 9、dismissed 1；产出锚新鲜非停产。
- 上游队列空：`fin_surface_gap` 全表仅 1 行 state='rejected'，
  pending=0——gap 驱动臂无料可消费属正常（面差枚举已分类完）。
- 转换链人工闸实证：adopted 卡→模板注册=显式批准径（run.go:8
  注释钦定）；`fin_opp_template` 12 行中与假想族对位者全
  state='candidate'+enabled=false（stake_carry/cmc/swap-carry-TRX、
  earn_term_spread、lending_spread 等）——启用与否=决策侧动作。
- 提案管道本身有活证：`fin_proposal` 最新行 2026-11-03
  （BTC-USD-270326 quarter_basis_long/perp_quarter_spread 族，
  enabled 模板评估臂产出，state=fin_retired=走完生命周期）——
  「评估→提案」机械链非断。
- 观察项（非断线，呈决策席）：adopted 卡 okx/savings-rate（11-02）
  至今无对位 fin_opp_template 行——批准后模板注册步滞留≈3 天，
  属批准径队列深度问题非代码病灶；proposed 卡 11-04 新鲜在队。

### 断线径修票建议

**无断线径——三径皆正常空转**（判据内/批准闸/能力闸设计内），
无需另立修票。两处非断线观察供决策席参考：①xvenue-carry 提案面
当前被「mt5 trade-order 能力面未启+execfam 未登记（T-0145 未派案）」
双闸钉死——若业主意图放行跨所 carry 提案链，需另立「mt5
trade-order 能力面启用评估+T-0145 执行径派案」票，非本审计范围；
②adopted 假想卡模板注册滞留=批准径队列深度，归决策席例行盘。
