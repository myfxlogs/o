---
schema_version: 2
id: order-leg-sz-unit-mismatch-naked-bag
kind: pitfall
risk: critical
status: verified
keywords: [fin_order_leg, sz_unit, quote vs base, 平仓腿单位错配, delta_drift 幻觉环, 裸袋, CORE, 账实背离, closing attempts, 量纲错配, TgtQuote]
scope: [平仓计划 qty 计算, delta_drift 守卫 notionalNow 漂移判官, fin_order_leg.sz_unit 消费面, fin_position/fin_holding/fin_guard_close]
summary: fin_order_leg.sz 语义随 sz_unit 双制（quote 额/base 量）——平仓计划与漂移判官拿 quote 额 sz 当 base 量：现货腿只清 1.3% 产 3813 CORE 裸袋在 Earn 烧费+pos33/34 幻觉 drift 杀环+settled −24.67 假亏账实背离；消费侧必须认 sz_unit 或一律读 qty_base
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: [施工席, 决策席]
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0419-平仓腿单位错配-CORE裸袋与账实背离根修.md
  - type: server_fact
    source: postgres
    query: SELECT id,note FROM ops_event WHERE id IN (3058,3060)——COS 升件+根因指针：开腿 sz=quote 额(TgtQuote=true,qty_base=1915.63 对)，下游 notionalNow 与平仓计划拿 leg.sz 当 base——名义 −19500bps 触发 abs 兜底，同数发卖单只清 1.3%
    observed_at: 2026-11-07T00:00:00Z
  - type: server_fact
    source: postgres
    query: fin_position#33 CORE-USDT-SWAP 20:00:25 开→20:00:40 平(15s delta_drift)；#34 closing attempts=5 活循环；fin_holding CORE spot_dust=1924.61+earn_flexible=3813.34
    observed_at: 2026-11-07T00:00:00Z
---

## 事实/症状

- fin_position #33 CORE-USDT-SWAP：开现货腿 sz=24.999 `sz_unit=quote`
  （$24.999→qty_base=1915.63 CORE 对）；平现货腿同数 24.99 被按
  `sz_unit=base` 发单——只卖 24.99 CORE≈$0.33，现货腿只清 1.3%；
- pos33 开仓 15s 即触发 delta_drift 平仓（两腿配平 1923 张 vs 1915 个
  本不该漂）；pos34 同因 `state=closing attempts=5` 活循环——
  开→幻觉裸→错单位平→产袋→drift 更深→再开再平→袋更厚，烧手续费的
  活循环；
- 账实背离：仓标 closed 记 `settled_amount=-24.67`（假亏——成本未回
  记账当损），资产实留 venue：fin_holding `CORE spot_dust=1924.61`
  +`earn_flexible=3813.34`（Earn 自动收息——资产在但=无对冲方向多仓，
  违 delta 中性章程）。

## 根因或策略解释

- `fin_order_leg.sz` 是双制字段——`sz_unit=quote` 时 sz 是美元额、
  `sz_unit=base` 时 sz 是币量，同一字段两义=隐契约；
- 平仓腿 qty 计算错径：目标美元额被直接当 base qty 发单；
- delta_drift 守卫 notionalNow（漂移判官）同病拿 leg.sz 当 base 比
  contract 数（或按库存全账算漂移——残袋让真对永远漂）——单位错配
  在消费侧多点复发，非单点 bug。

## 处置

- 钦定语义（T-0419 票）：平仓腿 qty 必须=持仓腿实得 qty_base（读腿/
  读持仓余额），禁拿开仓 sz 的美元额当 base 卖——对不上的平=错平；
- 通则：sz_unit 双制字段的全部消费面（平仓计划/漂移判官/名义额估算）
  必须按 sz_unit 分派或一律读 qty_base——判据「消费侧不认 sz_unit」
  grep 全谱对拍；
- 裸袋处置（3813 CORE earn 袋）归置属 DM 裁域：方案 A 卖袋还账（市价清
  记真损）/方案 B 留袋续息（明示接受方向敞口，风险上案）——票内探明
  后 DM 定径，本席不代拍；
- T-0419 根修票在飞（P1 钱径，业主令直批，t/0419-close-unit-mismatch
  @ /opt/o-wt/t0419）。

## 证据

- check: fin_position#33/#34 状态+fin_guard_close.trigger_kind+
  fin_holding CORE 行（2026-11-07 生产实证）
- result: #33 15s 开平(delta_drift) settled −24.67；#34 closing
  attempts=5；CORE spot_dust 1924.61+earn 3813.34 残袋在账
- check: fin_order_leg 开/平腿 sz+sz_unit 对拍（ops_event#3060 根因
  指针）
- result: 开 sz=24.999 quote/qty_base=1915.63 对；平同数按 base 发单
  只清 1.3%——单位错配实证
- ref: fin_eng_lesson#7、T-0419 票面病灶节

## 反证条件

- check: 消费面全谱按 sz_unit 分派/qty_base 后，新开仓对不再 15s 幻觉
  杀、平仓腿清 100%
- falsifies_when: 若根修后 drift 杀环仍复发，则病灶不止单位错配——
  条目照改不退（残袋计入漂移判官的另一臂须另案）

## 适用边界

- 钱径执行面 critical——凡消费 fin_order_leg.sz/任何双制量字段的判官/
  计划面同型适用；
- 裸袋归置（卖袋/留袋）不在本条处置域——归 DM 裁定席域按章程办。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a（候选=「裸袋计」：fin_holding 残袋行 qty×mark>阈
  计数——钱径钱面无适配 gauges.tsv 表现册，随 T-0419 收口评估补建）
- class: n/a（候选：淤积）
- query: n/a
- threshold: n/a
