---
schema_version: 2
id: probe-restore-scope-overreach
kind: pitfall
risk: critical
status: verified
keywords: [验证探针, 复位径, 资金隔离, Earn 抽回, 真钱测试, dust_buyback, 越界动钱, scope creep]
scope: [一切真钱验证/探针/回卷件, okx, mt5, 钱径工具]
summary: 探针复位段把「归集路径上现存的一切资金」当测试产物处理——业主停泊仓 2139.96 USDT 被当测试金抽回买成 PENGU；复位/归集必须先验证资金归属（本次测试金台账），禁按账户/产品「有多少算多少」
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: [okx-rest-boundary, okx-exec]
evidence_refs:
  - type: doc
    ref: tools/okxverify/main.go
  - type: doc
    ref: tools/okxunwind/main.go
---

## 事实/症状

okxverify mutating 谱复位段：为「尘币复位」调用 `SavingsRedeem(USDT, 全额)`+划转+市价买回——Earn 内 **业主原有停泊仓 2139.96 USDT** 被全额抽走买成 PENGU（$2145 模因币重仓）。起因链：① 前一轮 `savings_cycle` 扫描 bug（遍历时拿首个非零行 BTC 即判「无可赎 USDT」）→ 误判 Earn 只有尘；② 复位逻辑假设「Earn USDT=上轮测试停泊产物」未做归属核验。回卷成本：~$7（0.33% 点差+四程手续费）。

## 根因或策略解释

复位径把「账户/产品现值」当「本次测试造成的增量」——二者语义根本不同。测试金的边界是**本探针生命周期内创建的流转**（卖尘所得、摆入金额），不是「该产品里现在有多少」。存量资金从未参与本轮流转，抽它=越权动业主资产。

## 处置

- **测试金台账制**：mutating 谱内每笔资金移动记流水（额/方向/时刻），复位段只允许反向冲销台账内条目——台账外资金一律不可见
- 归集/抽回前先比「目标产品现值 vs 台账应存值」——差额是业主存量，禁动；earn/asset 大户做仲裁快照（期初读+期末比对）
- 金额参数用**绝对记账值**非「现值-ε」——`savingsUSDT()-0.01` 这类「有多少抽多少」式调用在复位径禁用
- 探针期初打印全资产快照留证（本例幸有此行才能快速定损）

## 证据

- check: okxverify -mut 二跑 + okxunwind 回卷
- result: Earn USDT 2139.96→0→PENGU 278765 持仓；回卷卖出 filled→Earn USDT 2132.91 复位
- ref: 本目录 exec-error-ambiguous-arbitrate-raw 同族（歧态势+越权双坑）

## 反证条件

- check: 复位径带台账校验后再跑同类场景
- falsifies_when: 台账制下仍发生存量资金被卷入——条目升级 critical 复犯

## 适用边界

适用一切会动业主资金的测试/探针/回卷件；只读探针无此坑。与「歧态势仲裁」互补：那条管单笔调用的结果不确定，本条管复位径的对象范围越界。
