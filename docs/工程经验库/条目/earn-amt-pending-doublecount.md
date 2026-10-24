---
schema_version: 2
id: earn-amt-pending-doublecount
kind: pitfall
risk: high
status: verified
keywords: [savings, amt, pendingAmt, 双计, Earn, 在途申购]
scope: [okx Earn 余额读径, 权益聚合, 对账面]
summary: /finance/savings/balance 的 amt 已含在途 pendingAmt——amt+pending 求和=双计权益虚高
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: [okx-asset-funding, okx-account]
---
# earn-amt-pending-doublecount —— OKX savings amt 已含 pendingAmt，amt+pending 求和=双计

## 坑
`/finance/savings/balance` 的 `amt`=**当前持仓总额（含在途 pendingAmt）**——实证 loanAmt 2028.60+pendingAmt 100.40=amt 2129.00。把 equity 算成 `amt+pendingAmt`=2229.40=**在途申购双计**虚高。

## 实证
PHASE 2026-10-24：救援卖出款 ~100.4 被扫进 savings 成在途申购，`refreshParkingSnapshot` 按 amt+pendingAmt 落 equity=2229.40（venue 实读 2129.00）→ T-0045 聚合对账 venue-ledger 差 +100.4 真告警。手工按 venue 实读改回 2129.00 后平。

## 反证条件
- venue 响应字段先求证「是否互相包含」再加总——OKX 家族字段（amt/loanAmt/pendingAmt、eq/availEq/cashBal）常有包含关系，禁直觉加总
- 注释自证陷阱：原注释「pending 为赎回在途仍属停泊款」假设了方向——在途**申购**时 amt 已含 pending，在途**赎回**时语义可能不同（施工前 docs-v5 核双向）

## 机械化
- T-0046⑦：equity=Σamt 单字段；赎回方向语义核证后再议分径。
