---
schema_version: 2
id: datedfut-leg-phantom-cashflow
kind: pitfall
risk: high
status: verified
keywords: [datedFUT, -YYMMDD, legCash, legPx, phantom spot_sell, sz_unit,
  qty_base, contractInst, 合约域判据, fin_cashflow, fin_order_leg, basis]
scope: [internal/trading/execution 腿账本落库面, internal/guard 平仓消歧面,
  OKX FUTURES 到期期货腿, basis 策略 datedFUT 腿]
summary: instID 后缀域判据只排 -SWAP 未排 -YYMMDD datedFUT——到期期货腿误落
  base 域：legCash 产 phantom spot_* 现金行（持仓期账本虚增）+legPx
  sz_unit=base 误标（qty_base 张数随 base 域成币量假读）；合约域判据
  必须单源枚举全形态（contractInst()），各处手写 HasSuffix 枚举=同族
  复发口；datedFUT qty_base=张数钦定——guard closeWant qtyBase>0 径
  消费锚（落 NULL=平仓量纲断链实证）
first_seen: 2026-11-08
last_verified: 2026-11-08
skills: [施工席]
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0483-datedFUT腿误分类-legCash-legPx账本失真根修.md
  - type: doc
    ref: docs/tasks/T-0481-basis-long-leg2败因诊断-同inst两格分流.md
  - type: doc
    ref: internal/trading/execution/sizing.go
  - type: server_fact
    source: postgres
    query: "fin_cashflow #757/#758（exec#1123）及 #116 对子（exec#1121 持仓期）——phantom spot_sell +24998.82 USDT / position_ref=37；fin_order_leg sz_unit=base+qty_base=张数同误标（T-0481 产侧取证在案）"
    observed_at: 2026-11-08T00:00:00Z
---

## 事实/症状

- `fin_cashflow` 产线 phantom 行：datedFUT 合约腿（BTC-USDT-YYMMDD）
  被 `legCash` 当现货腿记 spot_sell +24998.82 USDT（position_ref=37，
  exec#1123 对子 #757/#758；exec#1121 持仓期 #116）——持仓期账本现金
  虚增，全周期净额碰巧归零（非正确性救，是语义坏）；
- `fin_order_leg` 同误标：`sz_unit='base'`+`qty_base`=张数——张数在
  base 量纲标下成币量假读；判据改正后 sz_unit='contract'+qty_base=
  张数=同形合法（量纲由 unit 消歧——票 T-0483 钦定）。

## 根因或策略解释

- `flow.go` 两处 instID 域判据手写 `strings.HasSuffix(instID,"-SWAP")`
  枚举合约域——OKX 合约域乐器有**两种形态**：`-SWAP` 永续 +
  `-YYMMDD` 到期期货（datedFUT，FUTURES 族 quarter 形）。后缀枚举
  漏一族=整族误落 default/base 域；
- 判据散写=复发口：execution 内两点半径修（legPx szUnit 分派+
  legCash 豁免闸）；同族缺口在 `internal/guard`（closeWant/
  sizedCloseLegs/bareBags 三处 HasSuffix）**仍在案**—— datedFUT 平腿
  误走 SpotAvailBal/Spot 残袋域（T-0483 次生发现，另立票）。

## 处置

- 单源判据：`contractInst(instID)`（sizing.go）= `-SWAP` ∨
  `datedFUTRe(-\d{6}$)`——legPx datedFUT 分支落 sz_unit='contract'+
  qty_base=fillSz（实填张数：票面钦定，消费锚=guard closeWant
  qtyBase>0 径——落 NULL 实证 quarter_basis_long 格红：SzKnown=false
  →SpotAvailBal=FUTURES CtValCcy(USDT)余额当减仓量→absent 残留）；
  legCash 豁免闸接 contractInst，rollback/maker 径同函覆盖零漏；
- 存量失真行：判词提案=校正冲销（recon_adjust 反向行+腿行 UPDATE
  纠偏）归 DM 裁——T-0483 票内提案在案，本票零写产侧；
- **防复发律**：凡新写 instID 域判据禁手写 HasSuffix 枚举——走
  contractInst/域判据单源件；新乐器族上架=判据件一处扩域；
- **改判据前先查 qty_base 消费面**（guard closeWant qtyBase>0/
  `szUnit=='base'` 分支序）——列语义收紧会把下游靠误标载荷续命的
  径打断（TestMatrixBasisCells 消费面闭包谱实证逮住）。

## 证据

- check: `go test ./internal/trading/execution/ -run TestDatedFutLegContractDomain -v`
- result: 修前红（sz_unit=base qty_base=non-NULL）→修后绿（contract+
  NULL+cashflow 零增）；全包 21.5s 绿
- ref: T-0483（本票）；T-0481 次生发现①（产侧 #757/#758/#116 取证）

## 反证条件

- check: `SELECT count(*) FROM fin_cashflow c JOIN fin_order_leg l ON
  c.source_ref=l.id AND c.source_table='fin_order_leg' WHERE
  l.inst_id ~ '-(SWAP|[0-9]{6})$' AND c.kind IN ('spot_buy','spot_sell')`
- falsifies_when: >0——合约域腿再产 phantom 现金行=判据件失效或有
  第二条现金行写径漏接 contractInst（本条目失效须重写）。

## 适用边界

- OKX instID 命名律下成立（现货=CCY-CCY 无 -6位数字尾，FUTURES=
  -YYMMDD，SWAP=-SWAP）；venue 新增乐器族/换命名律=判据件须重校；
- mt5 腿走 venue=mt5 分支（lot 域）不经 contractInst——两域判据
  并行不互替；datedFUT 平仓 sizing/guard 域缺口不在本条目处置面
  （另票在案建议）。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a（候选=「合约腿 phantom 现金行计」——query 即上
  反证 SQL，阈=1：任何合约域腿 spot_* 行=病灶复发；注册入
  gauges.tsv 归 DM 裁——候选 sql_ro 淤积类随判词包一并呈）
- class: n/a（候选：淤积）
- query: n/a
- threshold: n/a
