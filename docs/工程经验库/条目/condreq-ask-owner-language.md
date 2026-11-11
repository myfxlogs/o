---
schema_version: 2
id: condreq-ask-owner-language
kind: pitfall
risk: medium
status: verified
keywords: [fin_condition_gap, 条件请求, payload, ask, 人话, 业主面, T-0081, feishubridge]
scope: [internal/platform/condreq, 条件请求写臂, 业主面文案]
summary: condreq 双约——①payload 必带 ask=业主白话一句话（写臂塞机器 verdict=业主收天书）；②owner_required 布尔键标业主域——0干预钦定真业主件只四类（出钱/出凭据/红线放行/人在场），executor/holding 默认会话可办不推卡，会话裁量件标 false 显式豁免
first_seen: 2026-11-11
last_verified: 2026-11-11
skills: []
evidence_refs:
  - type: server_fact
    source: postgres
    query: SELECT id,kind,payload FROM fin_condition_gap WHERE state='pending'
    observed_at: 2026-11-11T15:20:00Z
    server_revision: ""
    response_hash: ""
  - type: doc
    ref: cmd/feishubridge/gate.go:93
---

## 事实/症状

fin_condition_gap #154 executor 卡推飞书后业主报「完全看不懂」——
payload=`{"source":"T-0438","verdict":"通过——…armed→open闸fail-closed
全hold…矩阵产跑派T-0480施工…"}`，写臂（会话判词）把裁定原文当请求体，
无 ask 键无白话。桥件降档渲 verdict 仍是黑话判词密度。

## 根因或策略解释

请求盒 schema 只 CHECK kind/state 闭集，payload 自由 JSON——写侧零约定
「业主读面」字段。写臂自然顺手塞自己的判词（它手头就是 verdict），
业主面契约真空。T-0081 人话律盖了告警链（Humanize 件）没盖请求盒写侧。

## 处置

写侧约入 canon（本条）+桥件降档兜底（verdict 键入抽取序）：
- **condreq.Request 调用方必写 payload.ask**——对业主的白话一句话，
  三要素：要你做（动词）、何时做、不给/拖会怎样。例：
  `{"ask":"真钱彩排须你到场——armed 13张策略等开仓,彩排不过零开火","verdict":"…","ticket":"T-0461"}`
- verdict/ticket/ref 机器面字段照写不删——黑话挪括注尾保追溯
- 桥件 askFromPayload 抽取序 ask→note→reason→verdict→原文兜底
- **owner_required 业主域闸**（0干预钦定——真业主件=出钱/出凭据/
  红线放行/人在场四类）：capital/credential/permission/constraint
  默认推；executor/holding 默认会话可办不推,真业主件标
  owner_required=true；会话裁量件标 owner_required=false 显式豁免。
  机器可办件禁挂 pending 等业主——会话域自办或销案,勿把
  请求盒当工单队列

## 证据

- check: cmd/feishubridge TestAskFromPayload 六夹具（verdict 降档含）
- result: #154 类 payload 自此渲判词非裸 JSON 墙；新写臂按约带 ask
- ref: cmd/feishubridge/gate.go askFromPayload

## 反证条件

- check: 抽查新 pending 行 payload 含 ask 键比例
- falsifies_when: ask 键普及率=0 且无新卡投诉（说明卡面 verdict
  降档已够——业主接受判词密度；本约降级为建议）

## 适用边界

条件请求写臂通用（hunger/vol_sell/会话直写同约）。executor 类尤其——
「需要人执行」的请求无白话 ask=业主无从下手。告警链 kindSpec 域
已有人话表不受此约。

## 伴生量表（T-0332 强制）

- gauge_name: condreq_no_ask
- gauge_sql: SELECT count(*) FROM fin_condition_gap WHERE state='pending'
  AND NOT (payload ? 'ask')
- gauge_interval: 1d
- gauge_desc: 缺 ask 键的未决请求数——写侧约遵守面（新件应趋零,
  存量件随结案自然消）
