---
schema_version: 2
id: watch-anchor-source-mixing
kind: pitfall
risk: medium
status: verified
keywords: [看门锚, 席 KPI, 锚污染, provenance, 静默告警, 度量口径, 锚源混源]
scope: [internal/research, 看门/告警面, 产出度量]
summary: 看门锚与 KPI 度量若不按席/臂限定产出源（provenance 谓词），异源鲜卡可擡锚掩真静默——「全臂产出旺盛」与「本席零产出」观测上不可分；锚查询必须带席限定谓词+判别桩对抗验证（T-0121 返修实案 wk_all=7/wk_seat=0）
first_seen: 2026-10-26
last_verified: 2026-10-26
skills: [策略评估与仓位]
evidence_refs:
  - type: server_fact
    ref: "本机部署库 5433 实拍 2026-10-26：fin_opp_hypothesis 全表周卡=7 而 provenance=researcher 席卡=0——grammar 枚举臂历史卡撑亮席度量/锚"
  - type: doc
    ref: docs/tasks/T-0121（返修回填——锚 SQL+metrics 三计数加 provenance 谓词+TestResearcherSilentIgnoresForeignCards 判别桩）
---

# 看门锚/KPI 不分席=异源产出掩真静默

## 病灶形

T-0121 研究员席静默看门首版锚=max(fin_opp_hypothesis 全表 max(ts), 席
Beat)——语法枚举臂（grammar_enumerate）、假想官（hypothesis_officer）产卡
同表同构，任何异源鲜卡都擡锚→席真静默 14 天也不告警；metrics 三计数同病
（席周卡显示 7，实为臂产出）。**「系统在产出」与「本席在产出」两回事**——
锚必须锚席自己的产出。

## 判据挂接

- 锚查询带席限定谓词（本席 provenance / 臂名 / 表列级 owner），非全表 max。
- 度量同谓词收窄（席 KPI=席卡）。
- 桩测须有**判别桩**：异源鲜卡（ts=now）+本席锚超龄→仍须告警；并做对抗性
  验证（谓词临时剥离→桩红，还原→绿）——「锚失效分支反绿」=桩假绿最阴形。

## 处置

T-0121 返修三件：silent.go 锚 SQL 加
`WHERE evidence->>'provenance'='researcher'`；tools/hypo metrics 同谓词；
card 测试桩补 evidence+新判别桩。部署库复跑 metrics 周卡 10→2 归正。
