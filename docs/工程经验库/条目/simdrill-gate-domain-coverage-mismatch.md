---
schema_version: 2
id: simdrill-gate-domain-coverage-mismatch
kind: pitfall
risk: high
status: verified
keywords: [simdrill, fin_exec_drill, 毕业闸, probe→live, dcd_put, 覆盖域不匹配, fail-closed]
scope: [internal/trading/simdrill, internal/portfolio/lifecycle]
summary: 毕业闸全策略同待遇要求演习凭据，但演习面只覆盖 SWAP——dcd 系 probe 物理不可毕业；闸凭据消费域必须≤演习产能域。
first_seen: 2026-11-09
---

# simdrill 闸凭据域≠演习产能域

## 症状
`probe→live` 拦 `exec_drill_missing`，但对该 probe 跑 `simdrill -strategy=<id>`
直接拒：`演习面限 SWAP 键策略（funding-hedge 毕业域）`——闸要的凭据
演习机根本产不出来。dcd_put #287249 实例实证。

## 根因
`lifecycle/paper.go execDrillPass` 对**所有** probe 一视同仁查
`fin_exec_drill verdict='pass'`（正确——fail-closed 不偏爱），但
`drill.go Run` 的器械面白名单只有 `-SWAP` 尾——毕业闸消费域（全模板）
> 演习产能域（SWAP）。dcd 系器械永远卡闸，且表面看像「凭据缺」
而真因是「凭据不可产」。

## 修法
单策略径按 `template_ref` 分流：`-SWAP`→原双腿径；`dcd_put`/`dcd_call`
→`runDCDDrill`（矩阵径 `runDCD` 产线同码序：md_dcd_product 夹具→
ephemeral 策略行→dcdexec.Open(dcdSim)→hold 态检→Settle 到期结算
→exec/position 态迁 tx→OpenExposure 零残留）——DCD 无平仓 API，
SETTLED 即平仓语义，无哨戒段。凭据落账抽 `writeDrill` 两径共用。

## 判据
改闸/演习这类「凭据产消对」时，两域必须同表核对：消费端要的
每类凭据，产能端都要有对应器械径——否则 fail-closed 变 fail-forever。

## 实证
- `TestDrillDCDPutFullPath`：dcd_put 全径绿+凭据落账+SETTLED→closed 态迁。
- #287249 演习跑通后 `probe→live` 解拦（产线实证）。
