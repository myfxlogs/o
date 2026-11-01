---
schema_version: 2
id: md-funding-rate-firstsight-snapshot-not-settled
kind: pitfall
risk: high
status: verified
keywords: [md_funding, funding_rate, 口径, ON CONFLICT, 首见冻结, settFundingRate, current_period, 实时预测, guard, funding_flip]
scope: [internal/data/okxws/store.go, internal/guard/facts.go, internal/research/, md_funding 消费方全域, okx]
summary: md_funding.funding_rate 列实为「期初首见实时快照」非已结算值——store.go ON CONFLICT (venue,inst_id,ts) 仅刷 recv_ts 不刷值列，首见后冻结；OKX current_period 制下帧内 fundingRate=当期待结算实时预测（双采样实证帧间变动），真结算值在 settFundingRate（未采）——任何把该列当「已结算费率」读的判据（guard ① funding_flip 在内）判的都是期中快照
first_seen: 2026-11-01
last_verified: 2026-11-01
skills: [okx-market-ws, 生产取证]
evidence_refs:
  - type: doc
    ref: internal/data/okxws/store.go:79-83
  - type: doc
    ref: docs/tasks/T-0084-预测费率采集前置闸.md 取证附录（支 t/0084-forensic）
  - type: server_fact
    source: postgres
    query: "SELECT ts, funding_rate, recv_ts, to_char(recv_ts-ts,'DD HH24:MI:SS') FROM md_funding WHERE venue='okx' AND inst_id='ETH-USDT-SWAP' ORDER BY ts DESC LIMIT 10"
    observed_at: 2026-11-01T01:25:00Z
    server_revision: ""
    response_hash: ""
  - type: doc
    ref: docs/tasks/evidence/T-0084-取证卷宗-20261101.md
---

# md_funding.funding_rate=期初首见快照，非已结算值（settFundingRate 才是，未采）

## 事实/症状

- `store.go:79-83` funding INSERT：`ON CONFLICT (venue,inst_id,ts) DO UPDATE SET recv_ts=now() WHERE ...`——冲突时**只刷 recv_ts，值列不刷**；funding 帧 ts=结算拍（fundingTime），同拍内逐帧（实时预测逐帧变动）全撞同键 → **首见值冻结**。
- DB 亲证：ETH-USDT-SWAP 行 ts=10-30 08:00 拍、recv=10-30 02:56（结算前 5h03m 入库）——入库的是期中实时值；同期 venue 帧 fundingRate 逐帧变动（11-01 实采 -0.00001363→-0.00001411→-0.00001455）。
- 现行 OKX 帧（2024-03-05 改制后 current_period 制）：`fundingRate`=当期待结算**实时预测**；`settFundingRate`/`settState`=上期**真结算值**（未采）；`nextFundingRate` 死字段恒 `""`。

## 根因或策略解释

帧键=结算拍（T-0169 UQ 设计前提「funding 帧 ts=结算拍结算前恒值」在 2024-03 改制后**失真**——改制后 fundingRate 期内实时变动，非恒值）；冲突谓词（T-0199）为 recv_ts 鲜度设计，未顾及值列需随实时预测刷新 → 三重口径错位：funding_rate 列（标已结算）实为期初首见实时快照；结算时点终值预测无人捕获；真结算值（settFundingRate）未采。

## 处置

- 现状（T-0084 作废 obl122）：留码不启不删；已结算翻面径零改。
- 重飞/修口径径（备档供另票）：冲突谓词加值列刷新（收结算前终值快照）+ 语义改判（funding_rate=当期实时预测）+ 增采 settFundingRate 补真结算口径 + `num()` 空串→NULL（D-616 标缺律）。
- 消费方须知：guard ①「已结算翻面」读的实为期初快照（premium 平稳期≈结算值，波动期偏差无界）；research/funding、sigdist、opener 同列同病。

## 证据

- check: 卷宗二.3 SQL（recv 早于 ts 时序判别）+ 卷宗一实采帧（fundingRate 帧间变动、settFundingRate 恒）
- result: 首见冻结+实时口径实锤
- ref: docs/tasks/evidence/T-0084-取证卷宗-20261101.md；T-0084 取证附录 §三

## 反证条件

- check: 同结算拍内多帧落库后 funding_rate 值随帧更新（store 冲突谓词加值刷新后）；或 venue 帧内 fundingRate 改回期内恒值
- falsifies_when: md_funding 同 (venue,inst_id,ts) 行 funding_rate 出现 UPDATE 痕（非首见值）；或 OKX 官方回滚 current_period 改制

## 适用边界

- 适用 OKX perp（current_period 制）；MT5 面无 funding 概念（swap 走 md_signal kind='mt5_swap'）。
- 期初首见值与结算值偏差方向不定：premium 平稳期近似、波动期无界——禁止以「≈已结算」为前提做资金敏感判据。
- T-0084 票（作废）取证附录为全证据基座；退役/复用另票裁定（D-0042 十一月不换源不新基建）。
