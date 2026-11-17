---
schema_version: 2
id: fundccy-legcommit-toctou-residual-sweep
kind: pitfall
risk: high
status: fixed
keywords: [residual_clear, FundCcy, fin_position, fin_order_leg, TOCTOU, 杂币误扫, 开仓提交窗, sweepResidual, coinInFlight]
scope: [internal/earn/residual.go, internal/earn/residual_clear.go, internal/ledger/ledgered.go, internal/trading/execution/flow.go]
summary: 判据快照读早于写侧commit=在持集漏新腿币——earn sweepResidual拍头读FundCcy，而fin_position行在双腿成交后才落，腿filled→仓行commit窗内venue币已可动被误判杂币市价清（仓#86 VVV实证+24.26幻影）；凡「读集判可动性→对venue下单」两段式径，落单前必须重读判据集或按在飞commit信号让位。
first_seen: 2026-11-16
last_verified: 2026-11-16
skills: []
evidence_refs:
  - type: ticket
    ref: "工单#21723 residual-clear-fundset-toctou（升件#21721）"
  - type: server_fact
    source: postgres
    query: SELECT state,inst_id FROM fin_execution WHERE id=1213; SELECT leg_no,inst_id,state,qty_base FROM fin_order_leg WHERE execution_id=1213
    observed_at: 2026-11-16T21:05:00Z
  - type: commit
    ref: fix/residual-clear-toctou
---

## 事实/症状

仓#86（execution 1213，VVV-USDT-SWAP 对冲）spot 腿 272
（VVV-USDT buy 1.1989@20.26）filled 于 20:34:56.96；同 ts
ops_event 落「okx VVV 杂币敞口」升件，1.3s 后 decision#35285
residual_clear 市价卖 1.1977 VVV——腿币被自家清扫径卖掉，
腿账仍记持仓→mtm 幻影 +24.26、venue 已无币。

## 根因

`sweepResidual` 拍头一次性读 `FundCcy`（fin_order_leg JOIN
fin_position，**两表都要行**）+ `coins`（venue 现读），再逐币
处置。`fin_position` 在 `Execute` 双腿成交后才 INSERT
（flow.go）——腿 filled→仓行 commit 之间存在秒级窗：此窗内
venue 余额已含新币、FundCcy 尚不含。earn 与 execution 分体
进程无共享锁，纯时间窗竞态。

## 处置/纪律

- 判据集类守卫的通用律：**「读集→判→对外部下单」两段式，
  下单前必须重读判据集**（或持有与写侧 commit 同源的在飞
  信号让位）——拍头快照只配做粗筛，不配做放行令。
- 在飞 commit 信号=`fin_execution.state IN ('open','legged')`
  且币为其仓位/腿基币（coinInFlight 三径：exec.inst_id 基币
  ∨ venue 同域腿 inst_id 基币）；查败=保守让位（误放一拍
  优于误卖基金腿币）。
- 白名单停泊径同受害——park 也是钱动，在飞窗内同样禁行。
- 姊妹病灶：清扫造成的外部结算要在腿账留痕（fin_order_leg
  追加行挂真实 clOrdId），否则 mtm/diff 双吃（币值+现金流）。
