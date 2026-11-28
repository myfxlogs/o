---
id: ttl-verdict-on-venue-settled-product
title: 纯时龄判据用在 venue 到期自收产品形上=判词空转（零腿≠畸形须分形）
kind: pit
date: 2026-11-28
source: 工单 O-41392（值守卫生批 pos42 取证衍生——DCD put 仓 max_hold 空转实证链）
---

# TTL 判词空转：零腿产品形被当畸形腿报（O-41392）

**坑形**：`internal/guard` 仓守卫 `legs()` 把「查询成功且 0 行」与
「>2 腿/查询败/Scan 败」混进同一 `legSet{one:true}` 畸形桶——语义上
零腿有两种完全不同的成因：

- **零腿产品形**（DCD/结构品正常形态：exec 无 `fin_order_leg` 行是
  契约非病灶，venue 到期结算径 `dcdSettleOne` 收殓）——
- **真畸形**（>2 腿/读败）——aging 该继续报。

混桶后果：`decide.go` ③ `max_hold` 纯时龄判据对产品形空转——
`legs.one` 仓守卫径恒不可平（反向平物理不存在），每拍判「该平」
=语义错判。pos42 BTC-USDT-261204-66000-P 实证：exec 1134 零腿行
→ age=460h>240h → closeTrig{max_hold} → tend legs.one →
markLegAnomaly 定性行（幂等+节流，定性本身做对「需专道收殓」）——
DCD 产品期常 25d+，240h=10d TTL 意味着**每张 DCD 仓到期前必刷
~2 周空转判词**。

**根修判别式**（凡「空集」读面先问成因再分形）：

1. `legSet` 加 `zero` 位——**仅「查询成功 ∧ 0 行」**置位；
   `rows.Err()` 迭代败/Query 败/Scan 败/>2 腿一律不带（真畸形
   照报——**迭代败必须显式查 `rows.Err()`**，否则中途断流会
   伪装成干净零行=豁免误放）。
2. `decide.go` ③ max_hold 段 `legs.zero` 早退+`w.note` skip 行
   （复用 note 10min 节流键——「判词不适用」也须可观测）。
3. **只杀纯时龄那一条**——`legs.one` 的 strategy_degraded/
   retired 等事件判词照火（产定性行=正确簿记）；markLegAnomaly/
   tend 短路/dcdSettleOne 收殓径零动。

**夹具纪律**：存量测试若拿「即将被豁免的判词」当触发载具测下游
件（本案 TestLegAnomalyClassified/TicketSuppress 用 max_hold 载
零腿仓验 markLegAnomaly），豁免落地后须把载具换成仍合法的触发
（strategy_degraded 事件判词）——**验的是机制不是判据**，判据
语义变更不等于机制失效。
