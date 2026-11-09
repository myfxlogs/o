---
schema_version: 2
id: beat-arm-errc-fatal-amplification
kind: pitfall
risk: high
status: observed
keywords: ["errc", "log.Fatalf", "拍环臂", "ScanPoller", "md_tick_scan", "depblind", "观测盲", "臂级败", "进程死"]
scope: [cmd/collector, ScanPoller, 拍环臂, errc消费端]
summary: 拍环臂级可恢复败上抛 errc 被消费端 log.Fatalf=全臂死——T-0446 ScanPoller 一拍败杀死采集主进程姊妹臂陪葬；臂败=让位记拍，errc/fatal 留真岗终语义。
first_seen: 2026-11-0x
last_verified: 2026-11-0x
skills: []
evidence_refs:
  - type: doc
    ref: docs/工程经验库/待归纳.md（staging 原档——纪元3归纲转正）
---

## 事实/症状

拍环臂级可恢复败（全族 REST/目录瞬断、写败）上抛 errc，消费端 `log.Fatalf`=进程全臂死——T-0446 观测盲签名 `scan:md_tick_scan→discovery` 产者死径：ScanPoller.Run 一拍败即 errc→cmd/collector 杀死采集主进程，WS tick/sigdist/mt5 姊妹臂陪葬，md_tick_scan 停写→depblind 10min 零行窗报同签名。全仓仅此臂拍败即进程死（姊妹臂惯例=让位记拍下拍续试）。

## 根因或策略解释

臂级瞬断（可恢复）与岗终语义（真死）共用 errc/fatal 通道——瞬断被放大为进程级死亡，瞬断自愈能力被杀，且停写面触发下游观测盲签名误报。

## 处置

- 拍环臂败=让位不拖死主进程——臂级可恢复败走「记拍下拍续试」不走 errc/fatal
- errc/fatal 通道留给真岗终语义
- 持久真死由水位闸（scanwm 族谱）+探针（depblind 观测盲）升件收口——臂级瞬断不放大为进程级死亡
- 新增/审计拍环臂必核 errc 消费端语义——fatal 消费端下臂级可恢复败上抛=同病

## 证据

- check: staging 原档 T-0446 签名面病灶本体实录+全仓臂审计（仅此臂犯）
- result: 一拍 REST 瞬断→采集主进程死→多臂停写→观测盲误报
- ref: 待归纳 staging 原文（本条目转正件）

## 反证条件

- falsifies_when: 全部拍环臂 errc 消费端语义审计完毕且臂级败一律让位，或 collector 进程隔离使单臂死不拖主进程

## 适用边界

- cmd/collector 拍环臂族；姊妹臂同型 errc 上抛面 T-0446 时未全核（再现或全核坐实）
- 仅 Go 进程内臂间语义——跨进程/跨机不直接适用

## 伴生量表（T-0332）

- gauge_name: depblind 观测盲窗（md_tick_scan 零行计时）
- class: 断供
- query: SELECT max(ts) FROM md_tick_scan  -- 零行>10min=观测盲签名
- threshold: 零行窗 >10min 升件（depblind 在跑径）
