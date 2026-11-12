---
schema_version: 2
id: verdict-artifact-ordering-seam
kind: pattern
risk: medium
status: verified
keywords: [判物分离缝, 簿记先行, verdict_ref悬空, pendbr竞窗, organ探完工无票, 工单载支名, 工位断言, merge链挂出, dirty-stamp, 沿上采样]
scope: [ops_event 簿记面, 探针对账面, merge/部署链, canon裁定面, 工单覆盖约定]
summary: 簿记面（判定/票/断言/verdict）与工件面（文件/支/部署/戳）分帧落地时，先落簿记者恒引缝——缝窗内任一探针/对账沿上采样必报「半态」假阳或悬空引用。交付律=工件先于簿记或同帧双落；簿记永不得先于工件宣称其存在。
first_seen: 2026-11-12
last_verified: 2026-11-12
skills: []
evidence_refs:
  - type: server_fact
    source: postgres
    desc: "升件#11817：fin_eng_lesson#31/#34 已落 status='canon' verdict_ref=canon:两条目id，而条目文件仍在 fix/kb-lessons-r57 支未合——verdict_ref 悬空指向不存在工件；pendbr 探捞出后 merge 26a06238c 清"
  - type: server_fact
    source: postgres
    desc: "升件#11860：fix/selfnarr 推支→merge 同拍内 pendbr 探沿上采到「裸支无载名票」（工单#11385 note 未载支名=覆盖构不成）；发火瞬态命题为真、条件在解中——竞窗类假阳"
  - type: server_fact
    source: postgres
    desc: "升件#11867：工单#11385 处置行销后 OPEN 票零载词，organ 探报「pin 无票=悬置」；实=完工终态未落工位 done 断言——簿记（票销）先于工件（权威态断言）"
  - type: doc
    ref: "工单#11240 缝实证（散文接续+销案=链挂出，merge 工件活着而队列清零）；dirty-stamp 案（戳键 HEAD vs 测面工作树错配——评判工件与被评工件分离同源）"
related:
  - capgate-idle-refire-hold-verdict
  - probe-activity-anchor-pull-vs-push-silence
---

## 模式形

同一根因三案同窗显形（2026-11-12 单日内三件）：

| 案 | 先落的簿记 | 滞后的工件 | 缝被谁采到 |
|---|---|---|---|
| #11817 | lesson canon verdict_ref | canon 条目文件 | pendbr 裸支探 |
| #11860 | （工件先：支已推） | 载名工单 | pendbr 同拍沿 |
| #11867 | 工单处置行销 | 工位 done 断言 | organ 悬置探 |

更早同构：#11240 merge 链挂出（销案簿记先行、merge 工件未落）、
dirty-stamp（戳=评判工件，测面=被评工件，键位错配即评判失真）。

## 共性形

**判—物分离**：系统里所有「宣称 X 存在/完成」的簿记件（verdict_ref/
工单销词/工位态/绿戳/票载名）都与「X 本体」分两帧落地。两帧之间的
窗=缝；探针/对账/复审沿上采样到缝内任一侧，必然得到半态读数：
- 簿记先工件后 → 悬空引用（#11817 verdict_ref 指无物）或假「悬置」（#11867）
- 工件先簿记后 → 假「无覆盖」（#11860 裸支误中在飞链）

缝不是探针错——发火瞬态命题皆真；缝是交付序错。

## 规则（归一后逐例可查）

1. **簿记不先宣物**：断言「X 存在/已完成」的簿记行（verdict_ref/
   工位 done/销词载 sha）落笔前，X 本体必须已物化可查——文件在树、
   sha 在链、行在表。先宣后落=悬空引用债。
2. **覆盖名随票生**：码修工单落票即载支名/工件名——票是探针的覆盖
   凭据，名缺=工件裸奔（#11860 案：票未载 selfnarr=探不误发）。
3. **终态断言行同处置帧**：销票/销器官时，权威态断言（工位 done/
   部署行/复绿行）与处置行同批落——处置行是叙述、断言行是权威，
   分帧=探针对「pin+无票」半态采到「悬置」假阳。
4. **评判键=被评件**：戳/断言/对账的键必须指向被评工件本体非代理
   （HEAD≠工作树——dirty-stamp 案）；键错配=评判失真且静默。
5. **缝窗可抑不可灭**：推支→合流这类物理上必两帧的链，缝窗天然
   存在——抑法=簿记侧载名（票载支名）使探针有覆盖凭据可认，而非
   拉宽探针窗（拉窗=埋真缝）。

## 处置引径

升件定性含「簿记—工件分帧」嫌疑时，先问：簿记先落还是工件先落？
- 簿记先 → 悬空/假悬置：补工件或断言帧，勿改探针。
- 工件先 → 假无覆盖：补载名簿记（票/断言），勿改探针。
- 两判皆非 → 按件本级处置档走。
