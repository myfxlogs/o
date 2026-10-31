---
schema_version: 2
id: dual-log-file-offset-cross-contamination
kind: pitfall
risk: medium
status: verified
keywords: [headlesspump, 双文件票, offset 交叉污染, emitSoon 重发, run/audit 同 id, 对账拍, 全量重读, ops_headless 写放大]
scope: [cmd/headlesspump, 对账/轮询拍设计, 单 offset 状态机+多文件源, 写面节流]
summary: 单 offset 状态机喂多文件源（run/audit 同 ticket id 双日志）时，轮询对账拍轮流读件→offset 随 logPath 交叉污染→size<offset 误判「截短」→全量重读→emitSoon 无门控重发=每拍重写（实测 16 张 done 票 4 写/s）。修=对账拍只服现役件（logPath==f 才 tail），交叉并发仍走事件径；轮询拍引入前必查「同 id 多源」形态
first_seen: 2026-10-31
last_verified: 2026-10-31
skills: [生产取证]
evidence_refs:
  - type: server_fact
    ref: 修复前 journalctl emit 4/s 连环（16 张 done 票每 5s 重写，seq 全表水印 5/min→+516/30s）；修复后无在飞 0 emit，75s 窗 229 行仅 1 行有写
  - type: doc
    ref: cmd/headlesspump/pump.go tick① 现役件门控+TestTickDualFileNoThrash 回归锚
---

# 对账拍×双文件票：offset 交叉污染→重发连环

## 症状

泵加 5s 对账拍后，16 张终态（done）票每拍重写 ops_headless（journal emit 4/s 连环、seq 水印狂奔），而单文件票全静——写放大只打在同 id 双日志（devin-headless-T-xxx.log + devin-audit-T-xxx.log 并存）的票上。

## 根因链

1. tk.offset=单锚（设计假设：一票一日志一尾巴）。
2. ensure() 每次按调用覆写 logPath——对账拍轮流喂两件 → logPath 拍间翻转。
3. onLog 拿 A 件的 offset 读 B 件：size(B)<offset(A) → 「截短复位」offset=0 → 全量重读 B → 行锚重解析（态不变 setState 门控住）→ **尾部 emitSoon 无门控** → 每拍重写。

## 纪律（复用）

- 轮询对账拍引入前，先盘数据源形态：**同 id 多源件**（run/audit 双日志、轮转件、分片件）一律「只服现役件」（logPath==f 才 tail），源切换走事件径。
- 截短判据（size<offset）在全量重读路径上必须配 emitSoon 类节流尾帧的门控——重读≠活动。
- 观测锚：journal emit 频率 vs DB updated_at fresh 行数对拍——两者背离=写路径有「日志在响、行没在动」的幻觉源。

## 关联

- 状态面秒级化钉案（2026-10-31）：tick 三门控+防抖回归 TestTickDualFileNoThrash。
- 旁获未决：双文件票 seat 在 DM/Dev 摆动（ensure 覆写语义，事件径既有行为）——另票议。
