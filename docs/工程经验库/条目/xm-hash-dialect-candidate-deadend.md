---
id: xm-hash-dialect-candidate-deadend
title: venue 方言层未识→候选机零命中死路+前端无手输兜底=每 broker 接入卡人工
kind: pit
date: 2026-11-23
source: 工单O-35250（XM 目录 218 符全谱 #尾方言实证）
---

# 方言剥层漏一档=候选空死路（步④映射面实证）

**坑形**：`splitDialect` 两层剥（marker 档位标→quote 报价后缀）漏认
venue 特有变体标——XM 目录 218 符全谱 `#`尾=tradable 变体标
（BTCUSD#/GOLD#/US500Cash#，无#同名符=观察面并存）。`BTCUSD#` 剥不出
quote 尾→base 整体 "BTCUSD#"→base 等值全灭→候选零行；叠加前端候选空
无手输兜底（后端 SubmitSymbolMapping 本就能验任意 venue_symbol，UI
无输入径）→**每接一个新 broker 必卡死须人工下场**（业主令④接入自助化
语境下的设计缺口）。

**判别**：broker 新接入步④候选恒空但 SymbolList 枚举数正常——先查
方言剥层谱是否覆盖该 venue 全部后缀形态（#尾/.尾/连字符月年期货
BRENT-JAN27/指数 *_INDX#/*Cash#），再查前端空态有无手输+目录浏览兜底。
「后端能验≠前端能输」是独立的面缺口，须分开验。

**修法**：方言剥层按 venue 实拍谱补档（#尾=venue 可用标剥后续进
quote 层——剥层序 marker→#→quote）；候选空/截断时给自由输入
venue_symbol+目录浏览（rankCandidates 前 syms 原文如实给席位挑，
RPC 须回传原文字段）；canonical 下拉外自由文本同形闸。手输与下拉
同走硬验 RPC 单源——**禁替席决定，禁猜名直落行**。

**推广**：「剥层谱完备性」与「空态兜底面」是映射类 UI 的两条
独立验收线——只修一层仍会死在另一层。凡「机造清单→席位挑→硬验」
三段式，第二段空集必须有第三条人工径。
