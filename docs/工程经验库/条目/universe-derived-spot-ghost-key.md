---
schema_version: 2
id: universe-derived-spot-ghost-key
kind: pitfall
risk: medium
status: verified
keywords: [okxws, 宇宙扩件, UniverseCatalog, 幽灵键, ghost instId, MUU-USDT, 覆盖失明, 观测盲, funding-hedge, 现货腿派生, instruments]
scope: [internal/data/okxws, okx public instruments, okxws 订阅宇宙, hunger 腿覆盖探针]
summary: venue 衍生键归一须以目录实证为真值——UniverseCatalog 基币现货派生臂无条件入集，OKX 股票/新币 perp 无同名现货对（实测 305/525 幽灵），MUU-USDT 型虚构键入宇宙→订阅 error+探针假必报（T-0445）
first_seen: 2026-11-06
last_verified: 2026-11-08
skills: []
evidence_refs:
  - type: doc
    ref: internal/data/okxws/universe.go:144
  - type: doc
    ref: docs/tasks/T-0445-muu-usdt.md
  - type: server_fact
    source: postgres
    query: SELECT inst_id,template_ref,state FROM fin_strategy WHERE inst_id LIKE '%MUU%'
    observed_at: 2026-11-08T01:30:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

腿-行情覆盖探针（hunger ProbeLegCoverage）升件 `观测盲:MUU-USDT`：
live funding-hedge 策略腿所需行情键近 10min md_ticks 零新鲜行。
生产取证链：`MUU-USDT-SWAP` live 策略→`legset.FundingHedgeSpotArm`
剥 -SWAP 尾派生 `MUU-USDT`→宇宙扩件 `all` 无条件收→collector
tickers 订阅 OKX error 帧→refuseSub→md_ticks 永零→探针假必报。

## 根因或策略解释

派生臂语义假设「SWAP 基币币=同名现货对基币币」——OKX 股票/商品/
新币 perp 大量无同名现货对（MUU 实挂现货=`XMUU-USDT` 异基币币；
全谱实测 525 个 live -USDT-SWAP 中 305 个基币现货对不存在）。
`UniverseCatalog` 基币派生臂注记自承「兜底」语义但实现直入 `all`——
SPOT 族读成时目录真值已在集，派生臂只产生族外幽灵键。探针判据
「腿键∩订阅宇宙」本身正确：虚构键本应宇宙外（coverage.go 注记
「虚构键宇宙外不求供价不炸」），宇宙含幽灵=上游供给件病灶。

## 处置

派生键改暂存（`derivedSpot` 不直入 `all`）——SPOT 族读成=live
目录真值闸：实证在的派生键经 SPOT 族环同键覆盖，族外幽灵滤除；
SPOT 族读败=兜底径全量派生照旧（族级降级语义不动）。效应面：
collector valid/探针 subUniverse/提名 catalog 闸同读一供给件，
幽灵键出宇宙后三面同愈。legset 派生臂/探针阈/签名面零触。

## 证据

- check: `go test ./internal/data/okxws/ -run TestUniverseCatalogGhostSpotGate`
- result: 修前红（MUU-USDT 幽灵入集）修后绿；FamFail 兜底断言原样绿
- ref: T-0445 收口 commit（t/0445-muu-usdt）

## 反证条件

- check: 对拍 OKX `/public/instruments?instType=SPOT` live 全集 vs
  SWAP 基币派生集，若某 SWAP 基币现货对实证存在却被滤除
- falsifies_when: 出现「现货对 live 在册但 SPOT 族响应漏行」的 venue
  行为（分页/态滞），真值闸会误伤实证腿——届时需改按 instId 精确
  回查而非族并集判据

## 适用边界

- OKX instruments 公开面语义（state=live + instId 形）；他 venue
  若无 SPOT/SPOT 族分族面，真值闸需按各自目录形态另裁；
- 只治「宇宙含幽灵」——腿键归一正确性（MUU→XMUU 映射）与策略
  准入核现货对存在性属另域，不在本条目处置面。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a
（豁免由：复发信号=hunger 腿覆盖探针 `覆盖失明`/`观测盲` 升件行
——探针面即监测（24h 同键去重在案），gates.tsv 无独立表适配）
