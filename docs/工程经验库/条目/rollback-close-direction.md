---
schema_version: 2
id: rollback-close-direction
kind: pitfall
risk: critical
status: verified
keywords: [回滚, Close语义, 平仓Intent, reduceOnly, 方向反置, LEGGED]
scope: [cex 双腿执行, okx, 回滚径, 平滚方向逻辑]
summary: 回滚腿 Close 语义必须随原腿 Intent 反置——平仓 Intent 的回滚=重开原向仓非再平，无条件 Close=true 打在空仓上=拒单歧态势
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: [okx-exec]
---
# rollback-close-direction —— 回滚腿的 Close 语义必须随原腿反置：平回滚=重开非再平

## 坑
`rollback()` 无条件 `rev.Close=true`——对**开仓 Intent** 正确（回滚=平掉已开腿），但对**平仓 Intent** 灾难：腿1平仓单成交后回滚=**重开原向仓**，带着 `Close=true`→`reduceOnly`+平仓 posSide 打在空仓上=拒单→歧态势。

## 实证
PHASE exec5（2026-10-24 真户）：close 平仓腿1（buy 平 SWAP 空）filled，腿2 现货卖拒→LEGGED 触发回滚腿1。回滚单 ph5rb277315=sell 0.03 带 reduceOnly（此时空仓已无物可减）→ ambiguous；venue 实勘空仓=回滚实拒。净态=永续已平、现货裸持 0.00030866 BTC（$25 无对冲敞口）+ rollback_failed。

## 反证条件
- 回滚方向律：`rev.Close = !leg.Close`——开腿回滚=平，平腿回滚=重开
- 通用律：**反向单语义随上下文变，不是恒定值**——同 side 的反向在 open/close Intent 下含义相反，语义位必须按 Intent.Kind 派生
- 回滚歧义禁盲重试（51603 absent 才定态）——本案例 ambiguous 已是正解

## 机械化
- T-0046：rollback() 按 leg.Close 反置 rev.Close；测试锚：close Intent 的 LEGGED 回滚腿 Close=false+无 reduceOnly。
