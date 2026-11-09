---
schema_version: 2
id: pg-date-vs-timestamptz-bound-truncation
kind: pitfall
risk: high
status: verified
keywords: [postgres, date, timestamptz, 边界截断, 期首日, macro_events, report]
scope: [pg, report, metrics, 期界, 日期列]
summary: DATE 列对 timestamptz 界——PG 把界也截为 date，「End=今天」日行全排；期首日指标恒 0
first_seen: 2026-11-09
last_verified: 2026-11-09
skills: []
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0510-macro-events-date-trunc.md
  - type: doc
    ref: internal/ledger/report/report.go:101
---

## DATE 列×timestamptz 界=界被静默截回 date——期首日行全排、指标恒 0

- **实证**：`md_macro_event.event_date`（DATE）查询
  `event_date>=$1 AND event_date<$2`（$=timestamptz）——PG 对
  date×timestamptz 比较把**界截回 date**：`event_date < '2026-11-09 09:33'`
  等价 `< date '2026-11-09'`=当天行全排。周一 BoundsOf weekly
  Start=今 00:00/End=now → 窗内行全灭，macro_events 恒 0
  （TestReportMacroEventsCount 周一红，T-0484 挂起同锚）。
  探针实证：桩插两行在库 count=0。
- **波及**：每周一 weekly、每月 1 日 monthly 生产指标系统性失真——
  非测试 flake；且红一切 master 推（check_all 闸）。
- **教训**：DATE 列的期界查询，界必须同型化——`$1::date` 起界+
  `($1::date+期长)::date` 止界，勿裸 timestamptz 界混比；
  「期首日即 End 日」的窗是这类截断的唯一显形日——日常绿是
  假绿（边界天不在窗内才不显）。
