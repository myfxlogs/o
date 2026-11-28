---
id: candidate-family-eval-arm-gap
title: 骨架族登记≠评估臂——researchArms 硬编码枚举让 candidate 态族饿死
kind: pit
date: 2026-11-28
source: 工单O-41394（breed_stall 24h 内 3 火——三族 fin_opp_eval 全 0 行）
---

# 注册三件套与评估臂是两本账

**坑形**：新族走「骨架卡→candidate 态」径时，md 骨架迁移把模板登进
`fin_opp_template_roster`（std_state='candidate'），grammar.go 也登记了
Family 枚举——但评估臂在 `cmd/research/dcd.go researchArms` 是**逐族
硬编码实例**（funding/dcd/dirmom/dirtrend/trendmech/gridfit/gridcfd…）。
三件套登记齐≠有人产 `fin_opp_eval`：三族（funding-extreme-fade/
mark-last-disloc-fade/xsec-perp-mom-basket）登记即裁后 eval 全 0 行，
`breed_stall` 探针 24h 内连火 3 次——探针是对的，病灶在臂缺席。

**判别**：roster/grammar 册上有名无实——`fin_opp_eval` 按 template
计数=0 且族龄超 breed_stall 阈。先查臂枚举面（researchArms go 语句
清单）再查族定义面；「枚举≠评估」——grammar Family 只产卡不产 eval
（Existing=true 族 evaluate 跳过），hypothesis 只产假想卡，
stakecarry 只写 template='stake-carry' 定值。

**修法**：径甲=泛化「按 roster/卡态枚举待评族」评估件（治本——族即
注册即有臂）；径甲不在则径乙=仿既有臂逐族落 Runner（影子产出零执行
零提案径）。臂判据=卡面 falsify/ev_formula 可算口径机读
（app_config research.<fam>.*，fin 链播种 SG replica 同步），
direction 词表须两链同改迁移对（fin_XXXX+md_YYYY 全等 ALTER）。

**推广**：凡「注册表/清单驱动」的产出面，新增成员时先问消费端是
枚举驱动还是硬编码清单——硬编码清单=每个新成员都要手工接线，
漏接=静默零产出被下游探针当病灶报。candidate 态族的验收锚=
`fin_opp_eval` 行存在性，不是 roster 行存在性。
