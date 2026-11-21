---
schema_version: 2
id: forensic-declared-vs-physical
kind: pitfall
risk: high
status: verified
keywords: [浑水式法证, 宣告vs物理, 账实对拍, NAV成分劈, 孤儿仓, sim域混账, 审计读面, fin_cashflow]
scope: [ops_event, fin_cashflow, fin_position, canon, 探针面, 审计角色]
summary: 宣告面只被「想到的失实型」罩住——一日三劈实证：canon disabled 实为 enabled、滞留票面前提全假、「策略赚钱」叙述 vs 物理账（策略净额分位级）。法证五审：成分劈/孤儿劈/域界劈/读面劈/域混劈；劈差→器官上新牙非报告留档
first_seen: 2026-11-21
last_verified: 2026-11-21
skills: []
evidence_refs:
  - type: server_fact
    source: postgres
    query: SELECT kind,count(*),sum(amount) FROM fin_cashflow GROUP BY kind
    observed_at: 2026-11-21T12:00:00Z
    server_revision: ""
    response_hash: ""
  - type: doc
    ref: ops_event scope=forensic-muddywaters
---

## 事实/症状

宣告面（self_state NAV/注册表 canon/票面前提/判例行）自称诚实，
探针只罩住被想到的失实型——未想到的失实型零防御。2026-11-21
一日三劈实证：canon 写 o-audit.timer=disabled 实为 enabled 且在喂
复审链；cmcweb「支滞留」票面前提整个假（产线所跑 sha 早在
master）；「策略在赚钱」叙述 vs 物理账——funding-hedge 族全史
funding 实收分位级，策略净实现≈0，盈利全部来自业主现货腿。

## 修法

法证对拍五审清单（一次性过堂，劈到的差→转常驻探针条目）：

1. **成分劈**——不只对总数：NAV 实现盈亏按 kind 拆族，spot 腿收益
   ≠策略收益
2. **孤儿劈**——fin_position 开仓 vs strategy 活性：族已退役仓还在
   =无判官仓（trigger_verdict='orphaned' 机械可查）
3. **域界劈**——sim/演习行与真钱同簿时，查写径有没有闸（本仓曾
   纯靠消费面谓词——fin_0225 升库层闸）
4. **读面劈**——审计角色能否读到全部宣告底表（fin_ro 读不了
   fin_bot_*/fin_paper_fill=对拍径断，fin_0226 开闸）
5. **域混劈**——业主钱/服务钱/靓号钱/套利账同簿时，分域口径
   有没有机读件（fin_cashflow_domain 视图）

## 判据

- 「披露口径 vs 物理账实」每次法证必走 venue 真源（API/DB 原表），
  不接受中间汇总层自证
- 劈到的每个差=器官上新牙（探针条目/库层闸），不是报告留档
- 宣告盈利前先看成分：kind 拆解后策略域净额才是策略成绩单
