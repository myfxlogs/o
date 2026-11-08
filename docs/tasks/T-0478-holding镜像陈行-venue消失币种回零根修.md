# T-0478 fin_holding 镜像陈行根修——venue 消失币种缺席即删（幻影余额病灶）

- 状态：已批（CTO 立票直批 2026-11-08——T-0424 裁决附带发现病灶立票）
- 领取：无头施工席（o-worker-T-0478 @ t/0478-holding-absent-del）2026-11-08
- 优先级：P2——镜像面缺陷（幻影余额毒诊断取证/earn 汇总读面）
- 收口移交归：决策席

## 背景（T-0424 裁决实证）

fin_holding CORE 行：`spot_dust=3813.34977664`+`earn_flexible=3813.34977664`
stale=f asof=2026-11-07 10:59——venue 实拍 11-07 11:00 全卖后 CORE 余额
退出 venue 响应集，sync 只 UPSERT 不回零→幻影行永续 stale=f 挂陈值。

病灶：`internal/ledger/holding/sync.go` sync() 成功拍零缺席处置——
fetch 返回集外存量行永不收。

毒面实证：

- `internal/ledger/diagnosis.go` 取证查询（无 stale 过滤）——诊断
  草案拿幻影 3813 CORE 当 venue 实存；
- `internal/ledger/report/report.go` earn 汇总 `stale=false` 过滤虽在，
  幻影行 stale=f 照进——earn 停泊额虚增 ~$49。

## 钦定语义

```
修法面：成功拍收尾——fetch 覆盖域(venue='okx'×fetch kind 集)存量行
       不在本次返回集→DELETE（镜像=venue 实存真源非审计留痕；
       审计痕在 venue bills/ops_event 面另存）
护线：  只清 source='api' 且 venue='okx' 且 kind∈fetch 返回 kind 集行——
       bag.go 显形行/手工行（非 fetch kind 域）禁误收；
       fetch 败拍照旧全行 stale=true 不删（拉败非真消失）
```

## 范围

1. holding/sync.go 成功拍收尾加「缺席即删」段（护线谓词钦定）；
2. 桩测四断言：返回集缺 ccy→存量行删 / 同拍返回行照 UPSERT /
   bag 写行（非 fetch kind）不误收 / fetch 败拍全标 stale 不删；
3. 幻影行自洁：本修部署后下拍 fetch 即核销现存 CORE 幻影行——
   收口附部署后直查（自洁生效实证），不另跑手工 SQL；
4. 数据流.md 持账节同 commit 补「缺席即删」口径行。

## 验收

- 桩测四断言绿+check_all 对口；
- 部署后 fin_holding CORE 幻影行核销（行无或=0.0098 实值）；
- diagnosis/earn 报表读面无幻影。

## 结果回填

- 状态：待验
- 实现摘要：`internal/ledger/holding/sync.go` sync() 成功拍收尾加
  「缺席即删」段——fetch 覆盖域（venue='okx'×本拍返回
  (kind,venue_ref) 对）内 source='api' 存量行不在返回集→单句
  DELETE（unnest 参数化对集/键集，不在返回集=NOT EXISTS 反查）；
  fetch 败拍走上方早退照旧全标 stale=true 不删；fetch 空返回集
  （len(rows)=0）跳删——覆盖域不可判不核销。数据流.md 持有资产
  镜像口径行同 commit 补。
- **护线粒度偏离说明（审计要点）**：票面字面「kind∈fetch 返回
  kind 集」域会误收 bag.go 显形行——bag.go:149 写径
  kind='spot_dust' 与 fetch 同 kind（venue_ref='' 是唯一可辨面），
  kind 单维域下 bag 行必死=违「禁误收」钦定。实现按本拍返回
  (kind,venue_ref) 对收窄覆盖域：bag 行对域外天然不收、手工行
  source='owner_declared' 不收、structured_order 非 fetch 域不收、
  本拍未覆盖对（如 earn 面单独缺席）不连坐——比票面字面更严，
  非放宽。
- 测试证据：`TestSyncAbsentDelete` 四断言 PASS
  （O_TEST_PG_DSN=o-postgres-test:5438 真 PG，-count=1，
  testutil 独立 schema）——①覆盖域缺席行 GONE(spot_dust×asset)/
  GHOST(earn_flexible×savings) 核销 count=0；②同拍返回行 PENGU
  UPSERT 1.0→17.9 stale=false；③护线行 BAG(spot_dust×''=bag.go
  同形)/MANUAL(owner_declared)/NOX structured_order 播种行全存；
  ④fetch 败拍新铺 GHOST2 存且 stale=true（拉败非真消失）。
  `TestSyncAndStale` 回归 PASS。check_all 全链绿戳随支在案
  （pre-push 戳闸同 sha 校验）。
- 部署后自洁（验收锚②③）=待合并部署后执行——径：fin_holding
  表 CORE 行应无或=0.0098 实值（下拍 ev_account→sync 覆盖域内
  CORE 两行不在返回集→DELETE 自核销，不跑手工 SQL）；
  diagnosis.go 证据查询/report earn 汇总读面随之无幻影
  （stale=false 幻影行物理消失，读面零改）。
- 涉及面：internal/ledger/holding/{sync.go,sync_test.go}、
  docs/数据流.md。
- 范围外确认：未动 diagnosis.go/report.go 读面（幻影源在 sync
  写径——源修读面自愈）；未动 bag.go（其 venue_ref='' 写法即
  护线判据面，不改）；未跑手工 SQL；未碰部署面（合并+发窗归
  决策席常径）。

## 裁记（CTO 2026-11-08）

- 裁决：**通过**（merge 626e5b879 scoped 全链绿戳在案）。
- 先红证据（决策席亲跑补记——票面原未载，修前红律核查件）：
  merge-base 568833d8a 裸树植新测 sync_test.go 直跑
  `TestSyncAbsentDelete` → FAIL「缺席行 GONE 未核销 n=1」；
  修后同断言绿（施工四断言+本席 merge 树 scoped 复跑过）。
- 护线收窄偏离认可：(kind,venue_ref) 对域比票面 kind 单维更严，
  bag 显形行/手工行/structured_order 对域外天然不收——采信。
- 部署后自洁锚：本修入发窗批后下拍 sync 即核销 CORE 幻影行，
  届时直查 fin_holding 收尾验（批件随窗发）。
