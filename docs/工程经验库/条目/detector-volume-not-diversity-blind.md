---
schema_version: 2
id: detector-volume-not-diversity-blind
kind: pitfall
risk: high
status: verified
keywords: [探针判据, 多样性, 单因子, 假阴, mono_family, hunger, 量饱质荒]
scope: [internal/portfolio/hunger, 探针设计, 评估光谱面]
summary: 探针/饥饿判据只数绝对量不数分布——「量饱」掩「质荒」假阴：提案日产187判但97.9%单模板，hunger判「饱」而评估光谱已塌缩，369符MT5数据流三周零消费全程隐形
first_seen: 2026-11-20
last_verified: 2026-11-20
skills: [code-audit]
evidence_refs:
  - type: doc
    ref: internal/portfolio/hunger/hunger_mono.go（mono_family 判据——分布维补丁）
  - type: server_fact
    ref: fin_opp_eval 24h判词191件中187=dir_perp_mom（97.9%单模板浓度实数）
  - type: server_fact
    ref: md_signal kind=mt5_swap 369符在流∧fin_opp_eval venue=mt5=0（产消脱节实证）
  - type: doc
    ref: 工单#25649（多样性饥荒盲立案件）
---

## 病灶

判据形态 `count(X)=0→报警` 只覆盖「完全断流」——供给在流但**消费集中于单一对象**时计数器照常饱读，质量塌缩全程隐形。本案三处同病：
- hunger.Detect 只查「N天零提案∨paper池空」——187判/日照样单因子
- venue 产消无对账——MT5 数据流 3 周零 venue=mt5 判词
- 研究员四触发无「多样性饥荒」径——盲区靠业主点破才显形（业主令 11-20「这不应该由我来提」根因）

## 判据设计口径

分布类判据三件套：**浓度阈**（单对象占比≥pct）、**样本地板**（总量<N=噪声不判）、
**独立窗键**（判据窗与触发窗分离——7d判词窗 vs 3d零提案窗不同语义）。
样本地板=统计有效性下限不落配置键（非策略阈）。

## 教训

凡是「X 有没有」的判据，都要补问一句「**X 是不是只有一种**」——
绝对量判据天然漏分布病；供给量与消费多样性是两把不同的尺，产消对账单列。
