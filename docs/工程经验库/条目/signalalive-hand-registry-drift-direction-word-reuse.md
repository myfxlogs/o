---
id: signalalive-hand-registry-drift-direction-word-reuse
title: signalAlive 手工族谱漏收编=armed_ttl 全杀类病——判据须从通用产物推导非登记册；direction 词复用≠语义同族
keywords: [signalAlive, armed_ttl, 族谱, 漏收编, direction前缀, mt5_dir_mom, fundingEntryGate, 泛化判据, migrate_table派生]
---

# 坑：signalAlive switch 手工族谱随新族注册漂移——8 族 armed 4h 机械死刑

## 病灶（O-41370 三尸定案）

`signalAlive`（retire.go）存续判用 switch 手工族谱：vol 系/dir 系显式
分支收录，default 臂只认 `direction` 带 `short_perp`/`long_perp` 前缀
→fundingEntryGate，其余硬 false。eval 臂持续注册新族时族谱不补登记
=结构性漏收编：**mt5_dir_mom 78 尸 / grid_spot 12 / stake_carry 12 /
quarter_basis 系 112 / earn_term_spread / lending_spread /
perp_quarter_spread / grid_cfd**——armed 策略零执行侧活动+信号判死
→`armed_ttl_expired` 4h 准点收尸，毕业链（probe→live）另有
Drillable 断头叠加=永远毕不了业。

## 加深病灶：direction 词复用≠语义同族

mt5_dir_mom 的 `fin_strategy.direction`/`fin_opp_eval.direction` 词
= `long_perp`/`short_perp`（dirmom/persist.go 钉：direction 词沿用
perp 族位约定，**腿原语 `cfd_long`/`cfd_short` 才在 legs 证据面**）。
纯 direction 前缀路由把 CFD 动量误送 `fundingEntryGate`——该判词查
`md_funding` 费率同号，CFD inst 无 funding 源=恒 false=照死。
**「词复用」与「语义同族」必须分开判**——族属判据用模板册
（execfam.FundingHedge 唯一真源）非 direction 字符串。

## 修法（钦定泛化判据）

- default 臂：`execfam.FundingHedge(tmpl) ∧ perp 前缀向` →
  fundingEntryGate（判词本体零改）；**余向=dirEntryGate 同款 eval
  行存续判直调**（最新 fin_opp_eval proposed∧72h=活；行缺/陈/
  rejected/读败=死 fail-closed 一字不动）。
- 效果：8 漏族+未来一切 eval 臂新族自动免疫——族谱漂移类病绝根
  （migrate_table 派生哲学同判：O-41367 已把手工登记撞根改派生，
  本票判据从通用产物=eval 行推导，登记册不再是单点）。
- funding 族非 perp 向（xvenue cfd 副腿 dir=short_cfd）落泛化 eval
  判——行缺=死不虚构豁免（防「判词平凡真」变永生）。

## 教训

1. **手工族谱/白名单登记册=漂移源**——新族注册点有 N 处，漏一处即
   静默死径；判据该从「通用产物」（eval 行/族册/构型分类器）推导，
   让新族注册自动继承判据而非要求补登记。
2. **direction/状态词是族位约定非语义判据**——复用词表省 CHECK 约束
   扩容的代价是下游按词路由会误判族属；族属判定永远走模板/venue
   维度，词维度只做次级过滤。
3. **fail-closed 泛化≠总豁免**——泛化判据的死刑语义一字不动
   （行缺=死/陈=死/rejected=死），只把「无族谱注册」从死刑改候审；
   候审无证仍死=防永生闸不失。
4. grid_spot 毕业径同案根修：Drillable 断头（probe→live
   exec_drill_unshaped）须与存续判并修——单修存续判=产「永远毕不了
   业的僵尸 armed」；bot-drill 钦定径=simgrid 仿真 venue+产线
   Adapter 五端点同码（无腿可钻族以 bot 全径代腿钻，禁假腿凑闸）。
