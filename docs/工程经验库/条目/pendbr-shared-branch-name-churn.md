---
schema_version: 2
id: pendbr-shared-branch-name-churn
kind: pitfall
risk: medium
status: verified
keywords: [pendbr, 裸支, 支名复用, 并行会话, merge票, 接续棒, floor]
scope: [pendbr 探针, 工单/merge-* 接续环, fix/dispatch-pendbr]
summary: 并行 floor 会话共用同一支名 fix/dispatch-pendbr——每潮新 commit 推上去支尖即离 master 又成裸支,探针再火再立 merge 票:七波人工接续实证(#12136→#12894)。支本身无罪工件恒有效,病在「一支名多产线」与票接管节律错位。
first_seen: 2026-11-13
last_verified: 2026-11-13
skills: []
---

# 支名复用 churn——pendbr 七波同型坑

**实证谱**：`fix/dispatch-pendbr` 支被并行 floor-UI 会话持续推件，
每潮工件merge部署后尖再走远→pendbr探针再升件→人工立
`工单/merge-dispatch-pendbr-rN` 即并。已证七波
（#12136/#12212/#12292/#12346/#12389/#12894 +stuck_esc沿#12899），
r7 尖 7615df8b3 并入 o-core:0fa300f0056d。

**定性**：非宣称不实（每波判词皆载 merge+部署 rc=0 实证，
求助值守#12147 判读在案）——是节律性活产：工件有效、
支名复用、探针按「尖未入master∧无票载名」如实升件。

**处置径**（现行有效）：立 `工单/merge-<slug>` 载支名→票抑探针
+即唤接续棒→认领→merge --no-ff→check_all戳→push→deploy→
销票销升件。**裸销=盲区**，销后尖再走远下拍复火。

**根治方向**（未立件，留作设计债记录）：并行会话支名一潮一换
（`fix/floor-<slug>`）让 pendbr 各自归票，或工件面自带 merge
管道让机器自己并——外席每波人工接续=节律税非病灶修。
