---
schema_version: 2
id: sg-md-freeze-check-projector-truth-first
kind: pitfall
risk: high
status: verified
keywords: [o-projector, 八桥镜像, SG 冻结, PG-md 真源, md_signal, mt5_swap, carry_diff, 断供误诊, SG_PG_DSN, 127.0.0.1:5435, 双库对拍]
scope: [OCI o-collecto, OCI o-projector, SG md_* 可见面, 数据断供诊断序, FIX-02]
summary: SG 面 md_* 行冻结≠采集死——o-collecto 真写目标=OCI 本机 PG-md（SG_PG_DSN 实值 127.0.0.1:5435，env 名骗人），SG 全部 md_* 行经 o-projector 镜像；镜像僵死（重启后零日志）时 SG 断供而采集臂健康。断供诊断第一步=真源 vs 镜像对拍+projector 日志，勿先入采集侧病灶
first_seen: 2026-10-30
last_verified: 2026-10-31
skills: [生产取证]
evidence_refs:
  - type: server_fact
    source: postgres
    ref: 2026-10-30 16:28Z 对拍——PG-md：mt5_swap 13:11:50/carry_diff 13:11:50/SOLUSDm tick 16:28:25 全鲜；SG 同刻：mt5_swap 01:13:03/SOLUSDm 02:56:55 冻结 13.5h
  - type: server_fact
    source: rpc
    ref: o-projector 2026-10-30 02:56:57 重启，02:57:00 对账毕后 13.5h 零日志；重启前对账 173 inst/46 dcd 品→重启后 115 inst/0 dcd 品
  - type: doc
    ref: docs/FIX02-收口.md
---

# SG md_* 冻结=先查 projector 镜像，勿先入采集侧病灶

## 症状

SG 库 md_signal kind=mt5_swap/carry_diff（及 venue=mt5 SOLUSDm tick）末写冻结数小时——看似「生产数据断供」，健康闸红行指向采集臂（fix 提示词直指 MT5 桥 SymbolParams 层）。

## 根因

1. **拓扑认知错位**：o-collecto 的 `SG_PG_DSN` env **名**叫 SG，**实值**=`postgres://…@127.0.0.1:5435/o`（OCI 本机 PG-md）。SG 上所有 md_* 行的唯一来源=o-projector 八桥镜像。
2. **镜像单点僵死**：o-projector 2026-10-30 02:56:57 重启后 02:57:00 对账毕即静默（13.5h 零日志、零告警、零镜像），SG 面冻结而 PG-md 真源 4h 拍/tick 秒鲜全正常。

## 判据（复用）

- 对拍三查：① PG-md `max(ts)` per kind；② SG 同 kind `max(ts)`；③ `journalctl --user -u o-projector`（**--user**！系统级 journalctl 查用户级 unit 返回空，会误判「无日志」）。真源鲜+镜像冻=projector 病；真源冻才入采集侧。
- o-collecto 桥臂日志 `[mt5bridge]`（同 --user 径）：会话起/断+消哑 WARN 行。

## 误诊代价（本例）

FIX-02 急诊票按「桥 SymbolParams 断供」立单——探针三假设（mtapi 形态变/符号方言变/SG nil）逐一实测全否，桥无罪；白耗一个急诊施工席全程在排除正确组件。

## 关联

- docs/FIX02-收口.md（全证据链+移交件：projector 修复/健康闸推送径/25s OpenSession 互踩评估）
- bridge.go 消哑（FIX-02）：swap 全灭拍 WARN——采集侧再盲时 journalctl 必现形。
