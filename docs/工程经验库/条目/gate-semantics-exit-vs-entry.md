---
schema_version: 2
id: gate-semantics-exit-vs-entry
kind: pitfall
risk: critical
status: verified
keywords: [live_switch, kill_switch, close 径, 闸语义, 退仓, 拦平仓存风险]
scope: [internal/trading/execution, internal/risk, 全域闸件]
summary: 闸语义把退出当进攻拦——live_switch/kill_switch 一律 fail-closed 时关闸期有仓退不掉；开仓闸非退仓闸，close intent 免检（令牌照验不松）
first_seen: 2026-10-23
last_verified: 2026-10-23
skills: []
evidence_refs:
  - type: doc
    ref: internal/trading/execution/flow.go
  - type: decision
    ref: T-0021
---

# 闸语义把退出当进攻拦（T-0021 缺口一）

## 病灶

`Execute` 对 `live_switch` 一律 fail-closed——close intent 同走此闸=
**委员会关闸时有仓退不掉**。拦开仓=降风险，拦平仓=存风险，
闸的方向语义搞反：风险闸拦的是「新增敞口」非「削减敞口」。

## 修法

- Intent 加 Kind（open 缺省|close）：判序 open→live_switch fail-closed；
  close→跳过 live_switch（**令牌照验+留痕不松**——机械防绕过不降级）。
- 同族逻辑：kill_switch 不拦 close（T-0015 close 径已守）；任何新增闸
  先答「这闸拦的是进攻还是退出」——拦退出一律拒设计。

## 判据

新闸件 review 必查：close/降级/退出径是否被误拦；
fail-closed 判据缺省只对「增险方向」生效。
