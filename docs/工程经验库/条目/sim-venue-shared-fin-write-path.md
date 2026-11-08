---
schema_version: 2
id: sim-venue-shared-fin-write-path
kind: pitfall
risk: high
status: verified
keywords: [simvenue, simdrill, fin_position, 写径分域, 污染, postmortem, sim域, live域]
scope: [execution, guard, ledger, simdrill, dcdexec, notify]
summary: 仿真 venue 消费产线同码写径=仓行直落正本表——域界不是「记得加滤」，是构造期显式声明
first_seen: 2026-11-08
last_seen: 2026-11-08
evidence_refs:
  - type: ticket
    ref: T-0489
---

## 仿真/影子执行径与真户共用写径：域界靠构造签名强制，不靠下游过滤

- **实证**：simdrill 矩阵产跑仓直写 fin_position/fin_execution 正本（#35-39
  一秒五仓，全 fast_close）——postmortem 对 sim 仓立案空烧诊断额度、业主
  OKX 对账扑空、残仓判据腐蚀。产线同码是优点但「同码」含同写径=污染源。
- **教训**：仿真域借用产线消费件时，写域必须由构造参数显式声明
  （编译期漏接=错），禁止「可选参数默认真户」——默认即病灶同型。
  卫星表（order_leg/cashflow/guard_close）必须同簇分域：execution_id/
  position_ref 外键键空间共享即撞号回染真户读面。
- **通用解法**：域闭集类型 + 构造期 Must() panic + sim_ 命名域表镜像
  （LIKE 正本全形同序列 id 全域唯一）；聚合面（对账/日切/postmortem 立案）
  在 sim 域天然不跑，推送面双域选读各带域标。
