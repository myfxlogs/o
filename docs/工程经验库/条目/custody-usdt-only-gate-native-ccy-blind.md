---
schema_version: 2
id: custody-usdt-only-gate-native-ccy-blind
kind: pitfall
risk: high
status: verified
keywords: [custody对账, 币量对拍, USD折算, dcd_mtm_drift, 非USDT注资, DepositScanner, fundCcyOK, ledger_diff, BTC deposit]
scope: [ledger, funding, custody-reconcile, okx, DCD]
summary: custody 对账面按币量对拍——USD 折算豁免混入对账差+非 USDT 注资扫描闸双盲区（0.02BTC 注资永不入账+resolver 币量差恒0仍产$0假豁免）
first_seen: 2026-11-25
last_verified: 2026-11-25
skills: []
evidence_refs:
  - type: doc
    ref: migrations/fin_0247_phantom_audit.sql
  - type: server_fact
    source: postgres
    query: SELECT id,kind,ccy,amount,source_table,source_ref,occurred_at FROM fin_cashflow WHERE amount=1655.94
    observed_at: 2026-11-25T00:00:00Z
---

## 事实/症状

工单 O-36422 根查 okx_USDT 残差时实证双盲区：

1. **USD 折算混入 custody 差**：`dcd_mtm_drift` resolver 以「venue 名义量
   差×mark 现价」折 USD 作豁免额。生产实拉三分量真值：venue
   notionalSz=0.0207000000000000 冻结=基线凭据 0.0207 BTC——币量差
   恒 0，resolver 自部署起结构性盲算 $0。「0↔~49 打摆」表象=部署
   阶跃（静态锚 $53→跟踪 $0），|0−53|<2×53 破带带内静默采用——
   破带护栏按带宽放行是假安全。
2. **非 USDT 注资扫描闸**：DepositScanner/WithdrawalScanner 的
   `ccy!="USDT"` 闸使 BTC 注资永不入账（金源对簿记全盲）——
   depId=439722036（0.02 BTC，state=2，2026-10-21）漏账；后续
   DCD 申购腿（dcd_out -0.0207BTC）亦缺（手工基线凭据早于自动化）。
   BTC 面靠 $100 静态豁免硬压。mis-booking 本体=#33341 手工簿记
   USDT 面值，#33360 同日冲回——非代码径，但闸不修同类注资再漏。

## 根因或策略解释

custody 对账的判据是**币量守恒**（venue 持 0.0207 BTC 永是
0.0207 BTC，币量口径零残差）；USD 折算是 NAV 估值层语义（赚钱判据
统一计价需要）。两层混用=币量真值被折算噪声污染+豁免额随市价漂移
失真。resolver 跟踪值依赖 mark 价时，币量差为零仍产出非零假豁免
——「跟踪」名不符实。

## 处置

- custody 面纯币量：非本位键 ledger/venue 同币量比对，USD 折算只在
  容差比较时做一次（run.go reconcile 判据折算径已是此形）。
- 折算型豁免 resolver 退役——币量面能独立对拍的成分不需要 USD 折算
  豁免：`dcd_mtm_drift` 删注册表+`DcdOrderView`/`dcdOrd` 读件缝+
  venue_readers 接线三件套；豁免行 #18 expires_at=now()+reason 追加
  作废（fin_0244 先例，append-only）。`earn_accrual_gap`（USDT 本币
  应计）留任。
- 充提扫描闸=币形闸 `fundCcyOK ^[A-Z0-9]{2,20}$` 非 USDT 闭集——
  原生币按本域 ccy 落账零换算（ingest.go O-34287「各币各域」同形），
  reject 键 non_usdt_*→bad_*_ccy。
- fin_0247 历史补腿：fin_venue_bill dep-439722036+deposit +0.02BTC+
  dcd_out -0.0207BTC（配对手工基线凭据 id=165 缺腿），幂等双闸。

## 证据

- 迁移：migrations/fin_0247_phantom_audit.sql（豁免#18作废判词+
  BTC 注资三腿补账）
- 码面：internal/ledger/residual_tracked.go（resolver 注册表仅剩
  earn_accrual_gap）、internal/ledger/funding/{deposit,withdraw}.go
  （币形闸）、internal/ledger/run.go+cmd/core/venue_readers.go
  （dcdOrd 缝摘除）
- 生产实证：fin_cashflow id=1646 deposit +1655.94 USDT
  source_table=ops_event source_ref=33341（手工簿记）+id=1652
  recon_adjust -1655.94（#33360 冲回）；ops_event #33358/#33360
  溯源判词在案
- 桩测：deposit/withdraw 币形闸案（BTC 本域落账+形坏滤除）、
  退役键 fallback 案全绿

## 反证条件

- venue 侧出现「名义量随应计增厚」的 DCD 产品形态且币量差非零——
  届时币量对拍自然捕获真差，无需 USD 折算豁免（差额=真 custody 差
  应照报非豁免）。
- 非 USDT 注资属业主自营边界资产（非基金注资）——币形闸会入账，
  但 venue 侧同币余额同现→custody 对拍仍守恒；NAV 层若需剔除自营
  域另立口径（本条目不裁）。

## 适用边界

custody 对账面（ledger_diff/fin_known_residual 豁免设计）+充提
扫描记账径。NAV 估值层（统一 USD 计价）不在本条目射程——折算差
归该层是业主裁示原义。

## 伴生量表

- 对账面：ledger_diff_<venue>_<ccy> 告警频次（resolver 退役后
  USDT 面应无 DCD 折算成分复发）
- 簿记面：fin_cashflow 非 USDT ccy 行数（deposit/withdraw 原生币
  落账率——旧闸下恒 0）
