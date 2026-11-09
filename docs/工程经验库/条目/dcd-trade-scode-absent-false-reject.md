---
schema_version: 2
id: dcd-trade-scode-absent-false-reject
kind: pitfall
risk: critical
status: verified
keywords: [dcd, sfp, trade 响应方言, sCode 缺席, 幻影敞口, 业务拒误判, ordId]
scope: [internal/trading/dcdexec, internal/trading/okxexec]
summary: OKX sfp/dcd trade 成单响应只有 ordId 无 sCode——""≠"0"误走业务拒支=台账 failed 但 venue 仓真开（幻影敞口）；成交判锚=order-status 定真态非响应字段猜测。
first_seen: 2026-11-09
---

# dcd trade 响应 sCode 缺席=成单非拒——幻影敞口根

## 症状
#287249 dcd_put 摆渡成后开仓报「dcd 业务拒 : 」（码/文双空），
exec 标 failed 仓行缺；但 venue order-history 实证
ordId=11594773 LIVE 50USDT——**台账说败、venue 有真仓**。
若续试再成=重复敞口（台账永远看不见第二单）。

## 根因
`dcdexec.Open` 判 `rows[0].SCode != "0"` 走业务拒——OKX sfp/dcd
trade 成单响应=`[{"ordId":"11594773"}]` sCode 字段缺席，
unmarshal 出 ""≠"0" 误中拒支。仿真桩 `sCode:"0"` 喂出了
「成功响应必带 sCode=0」的假形——venue 方言只有真单实证能破。

## 修法
- 拒=显式非零 sCode（`sc != "" && sc != "0"`）；
- sCode 缺席且 ordId 在=放行，由 order-status 定真态（LIVE→open
  同 T-0160 律——成交判锚恒是 venue 查证非响应字段猜测）；
- ordId+sCode 双缺=ambiguous 禁补单（不可定态禁默认成功造仓）。
- 对账：幻影敞口 exec/position 行人工对拍修平（ordId=11594773 案）。

## 判据
venue 响应的**缺席字段**与**显式值**必须分别处理——`!= "0"` 对
空串为真是所有「code 类字段判拒」件的通病；成否判据永远落在
venue 查证端点（order-status/order-history），响应字段只做快径。
