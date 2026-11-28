---
id: sparse-stream-snapshot-verdict
title: 稀疏事件流上快照判据=把呼吸当死亡——连续窗才算消亡证据
kind: pit
date: 2026-11-28
source: 工单 O-41393（票#44861 dir-signal-flip-calibration——302 连杀实证：PIEVERSE long_perp 00:39 一条 rejected 行=杀 pos154 的那刀）
---

# 快照判据用在稀疏事件流上：一拍拒读≠信念消亡（O-41393）

**坑形**：`dirtrig.go` flip 段 `SELECT verdict ... ORDER BY ts DESC
LIMIT 1`——**最新一条 rejected 即杀**。但 `fin_opp_eval` 是稀疏
事件驱动（~日拍），市场态 dominant=volatile/chop 下评估面呼吸
正常——单条 rejected 行是「这一拍没过」不是「信念死了」。
302 连杀实证=谓词把呼吸当死亡：CRML/PONS 仓活证最新行 proposed
而旧仓已被一拍拒读处决。

**根修判别式**（「消亡/死亡/终态」类判词先问证据形态）：

1. **快照≠趋势**：事件流判据若源是稀疏事件（评估/探测/心跳），
   「最新一条=X」只证「最近一次=X」——须升级为**连续窗**：
   最新 K 条全满足才判（K 键化 `lifecycle.dir.flip_consec`
   缺省 2≈两拍距，dm 可写）。
2. **窗内异类即断**：任一行非目标态（proposed/unevaluable）=
   连续性断裂不杀——**「读不出」（unevaluable）不是证词**，
   禁计入连续链也禁单独构成杀证。
3. **样本不足不判**：`len(rows)<K`=历史独生行不杀——单读
   不构成连续（fail-closed 方向：证据不足宁可不杀，
   反向=误杀真仓）。
4. **K=1 幂等回退**：窗长可配 1=旧快照语义——校准件须留
   回退锚（机制验证+应急回滚径）。

**配套防坑**：`int(v)` 收 K 时 `v>=1` 闸——(0,1) 分数截断=0
会让 `len>=k` 对空集恒真=**空真杀**（valNonNeg 域收分数，
读侧必须自闸下界）。

**尸检可追**：判词 facts 载 `flip_consec`+`latest_verdicts`
连续链——死时窗内 verdict 序列留案，「为什么判死」可回放。
