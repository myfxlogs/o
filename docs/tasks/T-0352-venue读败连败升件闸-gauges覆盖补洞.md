# T-0352 venue 读败连败升件闸——gauges 覆盖补洞

- 状态：完成（决策席无头拍裁决通过 @2026-11-05——合 master@75fea746；三锚亲跑 PASS（合成连败跨阈恰一+日限/滞回清零/持续阈径——真库 1.29s/1.16s/1.34s）+check_all 本席复跑全链绿戳落 verify 树对口收口 SHA=755268ee+merge-tree 零冲突；锚③=部署后验项——o-core 部署后真 mt5 连败首拍升件行落 ops_event（seat=o-core event=升件 esc_class=读败 esc_scope=mt5，日限 1）｜原状态：待验（收口支 755268ee+收口后追记坑条目支 tip 151489ac 同合）；原已批 COS 例行代理批——业主令「工程部份还是做外派」定案）
- 领取：o-dev-t0352 @ 2026-11-05 08:31（业主令外派施工 run；T-0351 已由 08:05 前次 run 锁单施工中，本席只任本票）
- 级别：P3 可观测——venue 读面连败无人知晓，只刷日志=静默盲区

## 病灶（实查在案，勿再诊）

SG o-core 日志实证（2026-11-04 19:36–22:01Z+）：
`事件拍 fn 败: venue 现读: mt5 读败: mt5 facts/summary: account 空体 / <nil>`
约每 5s 一条、跨小时持续——CMC 公司账读面连败整夜，无任何升件/事件行。
读败只进 docker 日志=无人值守面事实失明（业主零可见性）。

同型风险不限 mt5：venue 读面（okx/mt5/其它）连败皆应升件——本票判据须
写成族通用件非 mt5 特例。

## 任务（修法供裁）

venue 读面连败→ops_event 升件闸，落点建议（施工可裁更优点位）：

- **判据面**：venue 读败计数已有内存/日志面——把连败状态喂进 gauges
  巡检拍（`scripts/meta/gauges_tick.sh` 族）或写 ops_event 失败计数行
  由 gauges 消费；
- **闸参**：连败 ≥N（建议 10 拍或 ≥5min 持续）+滞回（复常清零）+日频上限 1/venue——
  防刷屏三件套与既有升件闸同构；
- **升件行**：event='升件'，note 含 venue/败因摘要/连败时长——COS 收件箱
  三源之一即消化，业主可见。

## 验收锚

1. 合成连败（测试桩或干跑断言）→升件行落 ops_event；
2. 复常后滞回清零实证；
3. 真 mt5 读败现状下首拍：连败时长超阈即出升件行（天然锚——病灶在线）；
4. check_all 绿戳对口提交 SHA。

## 红线

- 不改 venue 读径本身（读败根因修另案）；
- 升件走 gauges/事件径，不新造通知栈；
- 判据族通用非 mt5 硬编码。

## 结果回填（O-Dev-t0352 @ 2026-11-05）

### 实现

- `cmd/core/venue_fail_gate.go`（新件）：`venueFailGate`——venue 键参数化
  连败闸。判据双径先到先升：连败 ≥10 拍（事件拍≈5s 面≈50s）或连败持续
  ≥5min；滞回复常清零（`ok()` 删态不延续）；日限 1/venue/日=`ops_event`
  `INSERT…SELECT WHERE NOT EXISTS` DB 侧去重（进程重启不破、复拍幂等）。
  升件行 `seat='o-core'`/`event='升件'`/`ticket=''` 空挂（不污染工单板）/
  `esc_class='读败'`/`esc_scope=<venue>`，note=venue+连败拍数/时长+败因
  摘要（200 字截断一行化）。观测面零侵入：fail/ok 任何败（含写行败）不上抛
  不拖读径——读径 fail-closed 原样（红线①内）。
- `cmd/core/venue_readers.go` 接线四面：`tradingVals` 包装（okx 权益=
  风险硬线+wd 守卫共享源）、`riskGate.WithEquity` mt5 支+okx
  `fundSaveBalances` 支、`ledgerR.WithBalances` 对账面（`tradingFundEq`/
  `fundSaveBalances`→okx、`mt5SummaryOnce`→mt5）——票面病灶拍径
  （`事件拍 fn 败: venue 现读: mt5 读败`，run.go:167 包装层）全覆盖。
- `scripts/meta/precedent_classes.tsv`：判类册追记 `读败 <venue> normal`
  （册律：新类随写随生防词面漂裂；esc_class/esc_scope 与 fin_0111 判例
  触发器匹配键兼容——同类判例命中可自动消化，未命中照常上行）。
- 落点自裁：票面建议「gauges 巡检拍消费计数/ops_event 计数行由 gauges
  消费」二选一；施工裁 **o-core 进程内闸**——连败态与读径同进程同拍，
  无跨面同步延迟；仍走 ops_event 升件既有径（红线②内，未新造通知栈）。
- 行数棘轮拆分（check-lines 硬闸逼出）：`venue_readers.go` 接线后 310 行
  >300 新超标——`okxAvailFromEvents`/`ctValReader`/`okxFundingAvailUSDT`
  三件 venue 现读辅助纯搬移至 `nav_live.go`（活读件文件，293 行；
  restlive.go 同例拆分先例在案），零语义改；`main.go` 一处文件指认
  注释同步。

### 验收锚对账

1. **PASS**——`TestVenueFailGateEscalateAndDailyCap`：合成 9 败零行→第 10
   拍升件行恰 1（note 三要素 venue/`连败 10 拍`/败因摘要实测断言）；
   `g.ok` 复常清零后同日再一轮 10 拍仍 1 行（日限 NOT EXISTS 去重实证）。
2. **PASS**——`TestVenueFailGateHysteresisReset`：9 败+ok+1 败=0 行
   （复常清零旧计数不延续）；`TestVenueFailGateDurationPath`：持续阈径
   （时长参量覆写）2 拍即升，venue=okx 非 mt5 特例。
3. **部署后验项**——真 mt5 读败现状下首拍升件行：闸在 o-core 进程内，
   须部署生效；病灶在线天然锚（mt5 facts/summary 空体连败现状持续，
   对账拍≈5s 面 10 拍≈50s 即出升件行，日限 1）。
4. 见下方机检行（绿戳=收口 commit SHA）。

### 偏离说明（供裁，未扩单）

1. **venue 级跨面共享计数**（票面未细分粒度）：对账/风控/守卫同 venue
   任一面读常即复零、任一面读败即累拍——venue 可达性语义；同 venue
   子面独败（如 savings 端点败而交易户读常）按设计不升件。
2. **覆盖边界**：earnings 读件（`WithEarnEarnings` savings 累计）与 earn
   径闲余读件（`okxAvailFromEvents`/`okxFundingAvailUSDT`）未入闸——
   venue 级聚合下其败被同 venue 读常面复零掩蔽，入闸=噪声非信号；
   主钱径读面（权益/对账）全覆盖。
3. **日限并发上限**：NOT EXISTS 为查询级去重，mutex 不跨 DB 写——同
   venue 并发双面同拍跨阈理论上限 2 行/日（后果=一条重复升件行，
   COS 消化幂等，防刷屏主目标不破）。
