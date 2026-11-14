---
schema_version: 3
id: paper-ttl-pos0-metabolism
kind: case
risk: low
status: verified
keywords: [paper_ttl, positions0, 设计代谢, 死法判词, admit_block适配界]
summary: retire_postmortem 簇 positions=0 TTL 到期=armed 闸在工作非死法——禁落 admit_block,判词口径结晶省每拍重裁。
first_seen: 2026-11-14
last_verified: 2026-11-14
evidence_refs:
  - {type: table_row, id: "fin_econ_lesson#149-204(dcd_put80+funding-hedge72+simdrill5)"}
  - {type: ops_event, id: "自问#15455"}
---

# paper_ttl positions=0 簇=设计代谢非死法（判词口径定案）

**情境**：retire_postmortem 165 案中 156 案为 dcd_put(80)/funding-hedge(72) 主簇，
全 `positions=0`——armed 期开仓条件恒未触发，paper TTL 到期轮转。
**实数**：dcd_put 80 案跨 ~40 个不同行权/到期键（每键≤2 次）=族提案面
广撒网，非同键循环死亡；现册 paper=1/live=1，占用无饥饿实证。
**根因**：paper 闸即防点本身——armed 期条件枯=闸正确拦截零入场，
零仓到期=练习位轮转设计代谢，非亏死非病。
**判据挂接**：可防一问=拦不住（防点在开仓条件闸内，TTL 是机制不是病）。
**回改点**：semantic:retire_postmortem判词口径——pos0∧ttl 簇裁定
verdict_ref 引 `canon:paper-ttl-pos0-metabolism`，免每案重推同判词
（#149-204 同批判例口径即本条前身）。
**证据**：fin_econ_lesson kind=retire_postmortem 簇表 positions/inst_id/
died_at 分布实拍；自问#15455 勘面题。
**适配界**：①**禁落 admit_block**——无仓零亏无组合死亡，禁键=冻结
健康族提案面，违「规约只写禁不写许」语义；②同键（template×inst）
反复 armed→pos0-ttl 出现=占用饥饿信号另立题（本簇未现）；③簇判词
不覆盖 positions>0 实亏案——有仓死亡仍按死法逐案裁。

**处置**：判词口径结晶本条；轮裁弃形不变（逐案 verdict_ref 行必落），
变的是引用源（canon 锚替判例链）。
