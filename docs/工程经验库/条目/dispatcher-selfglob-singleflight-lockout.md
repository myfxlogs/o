---
schema_version: 2
id: dispatcher-selfglob-singleflight-lockout
kind: pitfall
risk: high
status: verified
keywords: [systemd, list-units, glob, 单飞闸, 自锁死, 唤知机, dispatch]
scope: [scripts/cognition, systemd user units]
summary: 单飞闸 glob 匹配面含执行体自身 unit 名=dispatcher 恒退自锁死；闸面须剔自身。
first_seen: 2026-11-09
last_verified: 2026-11-09
skills: []
evidence_refs:
  - type: commit
    ref: 9052962c1
  - type: server_fact
    source: postgres
    query: SELECT count(*) FROM ops_event WHERE event='升件' AND ts>now()-interval '7 days'
    observed_at: 2026-11-09T14:10:00+08:00
    server_revision: ""
    response_hash: ""
---

# dispatcher 自匹配单飞闸=自锁死

## 症状
唤知机 o-cognition.timer 首拍跑 dispatch.sh，exit 0 零唤起——
信号面明明达阈（280 件未决升件），会话 unit 永不出现。

## 根因
单飞闸查询 `systemctl list-units 'o-cognition-*' --state=active,activating`，
而触发拍自身 unit 名 `o-cognition-dispatch.service` 也中该 glob——
dispatcher 在飞时把自己算作「在飞会话」→ 恒退。自锁死。

## 判据/修法
单飞闸 glob 必须剔自身 unit 名（grep -vE 'o-cognition-dispatch\.service'），
或闸面收窄到会话件命名形（`o-cognition-[0-9]*`）。
凡「查同名族 unit 判并发」的闸，先问：我自己在不在 glob 里。

## 教训归谱
同类闸面病：锁/闸的匹配面含执行体自身=必然死锁或假阳性。
纪元2 ops_headless 心跳按票分行同理（分行=剔自干扰），
纪元3唤知机踩的是同族坑——并发闸自检第一项=匹配面剔自。
