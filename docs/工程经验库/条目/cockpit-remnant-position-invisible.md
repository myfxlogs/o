---
schema_version: 2
id: cockpit-remnant-position-invisible
kind: pitfall
risk: medium
status: verified
keywords: ["残仓", "幽灵仓", "在仓面隐身", "策略退役产品活", "dcd_call", "T-0105", "驾驶舱", "zombie position"]
scope: [web/驾驶舱在仓面, fin_position, dcd/定期类产品]
summary: 策略 retired → 底层不可提前平仓产品仓（DCD 等定期类）仍在持仓，但在仓展示面随策略死而隐身——业主在 venue 总览见到活仓、驾驶舱查无此仓；账面结算 venue_bill 兜底无险，纯展示面洞，同类器械（期权卖方/定期 earn）退役后同型复发
first_seen: 2026-11-04
last_verified: 2026-11-04
skills: []
evidence_refs:
  - type: prod
    ref: OKX 总览持仓 DCD BTC-USDT-261225-95000-C 估值 $1592（2026-12-25 到期，业主实拍）
  - type: doc
    ref: docs/handoff/dialogue.md 2026-11-04 段（残仓实证录）
---

## 现象

DCD 试验族退役残仓 `BTC-USDT-261225-95000-C`（$1592、2026-12-25 到期）
在 OKX 总览仍为持仓中，O 驾驶舱在仓面不可见。T-0105「持仓列表全集化」
验收过全集，但「策略死/产品活」幽灵态滑出验收口径——在仓面派生源=
live strategy 记录，策略一退仓位展示随之消失，产品实物却还在。

## 根因或策略解释

在仓展示面从「策略存活」派生而非「position/产品行（venue 实拍）」派生。
可强制平仓器械两腿卖完=实物归零，策略死≈仓死，洞不显；**不可提前平仓
器械（DCD 定存、定期 earn、未来期权卖方）策略退后实物续存至到期**——
派生源错位显形。账面不受影响：venue_bill/cashflow 是 venue 真相驱动，
到期结算照入。

## 处置

类级修法（转决策席立案）：在仓面改从 position/产品行派生，或最低配增
「strategy retired + product open」remnant 类行展示。判别式：**有不可
提前平仓器械在册时，展示面不得以策略存活为存在谓词——以 venue 实拍
持仓行为谓词**。本具残仓 12-25 到期自清算，零操作。

## 证据

- prod: OKX 总览 BTC-USDT-261225-95000-C 持仓中 $1592（业主 2026-11-04 实拍）
- doc: T-0288 生死定性（dcd_call 策略全 retired 244 在案）→ 策略退≠仓灭
- doc: T-0105 持仓列表全集化（完成）——幽灵态未入验收口径

## 反证条件

- check: 若在仓面钦定为「只展示可管理仓位」语义（remnant 属刻意不展），
  本条目失效——彼时须票面明钉该语义边界
- falsifies_when: 决策档钦定在仓展示谓词=策略存活
