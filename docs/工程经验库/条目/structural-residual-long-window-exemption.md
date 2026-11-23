---
schema_version: 2
id: structural-residual-long-window-exemption
kind: pitfall
risk: medium
status: verified
keywords: [构性残差, 豁免到期复燃, fin_known_residual, 长窗豁免, 定性词, DCD市值漂移, Earn应计时基, diff_brief全成分]
scope: [internal/ledger 对账豁免面, fin_known_residual 簿记, 账实差告警初诊]
summary: 构性残差（市值漂移/时基差类——非簿记错、随市况缓漂）用短窗豁免=到期复燃循环：豁免#12-16逐代到期即升件再立案（okx/USDT残差族#6-#16六代实证）。正解=构性件长窗期（30d+）+reason带「构性残差-<成分名>」定性词+成分级分行；真差仍由未解释余量照报，豁免不夹带未定性差。
first_seen: 2026-11-23
last_verified: 2026-11-23
skills: []
evidence_refs:
  - type: server_fact
    source: postgres
---

# 构性残差豁免须长窗+定性词——短窗=到期复燃循环

- **坑**：工单#35226（residual-rootfix）。okx/USDT 账实差残差族
  豁免#6→#16 六代短窗续命，每代到期即升件复报——#16(dcd金源口径差
  $67.23)11-22 到期、#13(Earn应计$9.88)/#14/#15 11-23 到期，差额
  $62.89 复燃。豁免是续命非根修，差源在簿记层该销。
- **律**：残差先逐件定性再豁免——构性件（DCD 产品市值随 BTC 漂移
  口径差、Earn accrual×settle 时基差、venue 读时序瞬态）=长窗
  （30d）+reason 首词「构性残差-<成分名>」+成分级分行各豁其额；
  可根修件（幻影腿/谓词错）走码面销源不靠豁免；未定性余量不进
  豁免照报。
- **配套**：diff_brief 复报须含全成分——净额构成五件（现金流/
  停泊/dcd基线/原生基线/mtm）+在账豁免逐件 reason，差额可逐件
  归名非裸差（internal/ledger/diff_brief.go）。
- **证**：fin_known_residual #18/#19（工单#35226 长窗构性豁免）；
  ops_event 处置行分解落案（DCD~$52.89/Earn~$10/VVV=$0归名/余量
  ~$0-3）；VVV 成分核实=pos#86 closed→mtmReader `p.state<>'closed'`
  已排无需修。
