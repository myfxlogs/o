# T-0355 敞口闸读面根修——产品仓计入 exposure_cap

- 状态：完成（决策席无头拍裁决通过 @2026-11-05——合
  master@132fe760；收口支 tip=d6212c59，check_all 本席 verify 树
  复跑全链绿戳对口（专 DSN 5437）；票面 add/add 冲突取收口支
  完工回填版；三码面锚本席复跑实见；部署后验=首张 open intent
  的 fin_risk_decision 留痕见 dcdUSD 入 tExp｜原状态：待验）
- 领取：devin-t0355 @ 2026-11-05（业主令外派复工 run）
- 级别：P2 风控语义——闸漏看真仓位：DCD 0.0207 BTC（~$2.2k）在
  exposure_cap 判据里零计，新单压线时闸会按欠账放行=超真敞口

## 病灶（实查在案，勿再诊）

T-0351 工人回填标的孪生洞（其披露③）：`internal/risk/check.go`
`exposure_cap` 闸仍读 `fin_position` 单账——产品仓（dcd_baseline
凭据族）不入判据。显示面已由 fin_position_view 统一（T-0351 合），
**闸面没跟**——同一仓位显示可见而风控不可见=口径分裂的最坏形态。

## 任务（修法供裁）

- `exposure_cap` 敞口计口径对齐统一持仓视图（fin_position_view 同径
  或 dcd_baseline 凭据 Σamt 加项——施工按现构裁最贴点位）；
- dcd 币量→USD 估值同 cockpit 径（最新 tick 折算，valCcy 同径）；
- **tick 缺价=fail-closed**：闸面估不出敞口=不许放（与显示面
  「无 tick=0 不虚计」相反——闸面保守取，读败/无价宁可拒放，
  语义钉票不许取显示面口径）；
- exec 仓原口径零动。

## 验收锚

1. 合成 dcd_baseline 凭据行→exposure_cap 敞口值含其 USD 估值
   （测试断言）；
2. 缺 tick 行情下闸面 fail-closed 实证（拒放非零计）；
3. exec 仓敞口计数零回归；
4. check_all 绿戳对口提交 SHA。

## 红线

- 不改闸参/阈数值/decision 语义——本票只改**敞口读径口径**；
- 不动持仓/凭据写径；venue 全零 mutating；
- fin_position_view 消费面已有先例（T-0351）——同源不新造。

## 结果回填（devin-t0355 @ 2026-11-05）

### 实现

- `internal/risk/check.go`：open 径 venue_concentration 判后、
  exposure_cap 判前新增 `dcdExposureUSD`——取数走
  `fin_position_view WHERE kind='dcd'`（票面首选径=统一持仓读面同源，
  视图内建 `decision='allow'` 过滤与 (venue,ccy) Σamt 聚合零重造）；
  币量→USD 估值逐行：USDT/USD 名义直入，余币按 `ccy||'-USDT'` 查
  md_ticks/md_mark_index UNION 最新档（与 cockpit_face.go:110-118
  逐句同径，valCcy 同口径）。
- **fail-closed 钉死**：视图读败/行扫描败/无 tick 行/NULL 价/非正价
  任一→`ok=false`→`deny(exposure_cap)`——不取显示面 COALESCE(px,0)
  零计口径；deny reason 复用 `exposure_cap`（fin_risk_decision.reason
  CHECK 闭集枚举内，无新拒因=零 decision 语义变更，rejects 注册表零动）。
- exec 仓原径零动：fin_position+fin_execution 两查询逐字未改，
  dcdUSD 仅作加项入 tExp；close 径整块跳过（敞口只减不增语义不变）。

### 验收锚对账

1. **PASS**——`TestDcdBaselineCountsTowardExposureCap`：合成
   dcd_baseline allow 凭据 BTC 1.0+tick 50000→`dcdExposureUSD` 直断
   =50000；mark_index 单源径同值；闸级边界卡 `max=cur+50099`→
   deny exposure_cap（dcd 不入则放行的判别构造），放宽 1100→allow。
2. **PASS**——`TestDcdBaselineUnpricedFailsClosed`：无 tick 币 ZZX
   凭据→deny exposure_cap；混合仓（有价 ZZX+无价 ZZY）同拒；
   tick last=NULL 同拒；价复位→同凭据 allow（拒因=无价非他因）。
3. **PASS**——exec 仓零回归：既有 7 测全绿（allow/kill_switch/
   missing_param/token/close 族），exec 两查询零改动；
   `go test ./cmd/core -run 'Dcd|Untradeable|Close'` 12/12 绿。
4. 见机检行（绿戳=收口 commit SHA）。

### 测试夹具传导（判据扩面副作用——如实申报）

迁移种子 dcd_baseline（fin_0043 BTC 0.0207）入每个 Migrate 测试
schema→凡期望放行的夹具须配价+阈：`internal/risk/check_test.go
setup()` 加 BTC-USDT tick upsert；`cmd/core/open_dcd_reject_test.go
dcdRejectFixture()` 加 tick+`risk.max_exposure_usd` 放宽（种子阈
1000 被 dcd 估值 ~2070 顶破）。坑条目同 commit 入库
（gate-readface-widening-seed-credential-fixture-cascade）。

### 边界说明（供裁，未扩单）

- `venue_concentration`（vExp）仍读 fin_position+fin_execution 单账
  ——票面钉 exposure_cap 单点，本票未动集中度判据；dcd venue 归属
  （现全 okx）若需入集中度分子=另票。
- dcd amt=0 净额行仍要求对应币有 tick（净仓零估值也走价查）——
  保守向拒放，与「估不出宁可拒」同向，实际场景近无（基线行=部署
  凭据非对冲腿）。
- 生产验证锚：部署后首张 open intent 的 fin_risk_decision 留痕
  可见 dcdUSD 已入 tExp（本票合成断言已证机制，真仓读数=部署后验）。
