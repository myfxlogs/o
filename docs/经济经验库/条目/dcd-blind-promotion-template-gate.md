---
schema_version: 3
id: dcd-blind-promotion-template-gate
kind: case
risk: high
status: verified
keywords: [dcd, promotion, 模板盲, inst构型, 污染清扫, 判据挂接]
summary: DCD产品提案被模板盲promotion误建成funding-hedge paper——40+尸同根因,防点=promotion闸inst构型核(工单#15943)非开仓闸。
first_seen: 2026-11-14
last_verified: 2026-11-16
evidence_refs:
  - {type: table_row, id: "fin_econ_lesson#870-909(retire_postmortem簇strategy#2501-2545)"}
  - {type: ops_event, id: "工单#15943"}
---

# DCD 产品被模板盲 promotion 建成 funding-hedge paper（族级污染簇）

**情境**：retire_postmortem 40+ 尸（strategy#2501-2545 谱）同一死因——
DCD 产品提案被 promotion 模板按名匹配误建成 `funding-hedge` paper，
armed→TTL 到期 `positions=0`/`execs=0` 成批退役，从未开仓。
**根因**：promotion 闸只匹配模板名不核 inst 构型——DCD 期权构型
（期次/行权产品）被当资金费对冲族后代收养；D-0018 scope 令
（仅 USDT 计值品在域）未进 promotion 判据=制度缝。
**判据挂接**：可防一问=promotion 闸 inst 构型核（工单#15943 票面即此）——
提案进 paper 前闸件须验证 inst_id 产品类型∈族注册构型面；
launch 闸拦不住因提案本身合法只是错族收养，错配发生在 promotion
决策点非开仓点。
**回改点**：semantic:retire_postmortem 判词口径——同 template×venue×
reason(DCD盲promotion)簇裁定 `merge:ticket:15943`→`canon:`本条，
免每案重推同判词。
**证据**：fin_econ_lesson #870-909 簇表 numbers.reason 全文一致；
工单#15943 promotion-template-gate=码面根修载体。
**适配界**：**不落 admit_block**——规约键全等匹配无通配，
DCD inst_id 期次谱（BTC-USDT-*-P）枚举=漏网谱；真防点在 promotion
码面校验非开户闸黑名单。
