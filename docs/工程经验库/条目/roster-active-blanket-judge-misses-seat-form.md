---
schema_version: 2
id: roster-active-blanket-judge-misses-seat-form
kind: pitfall
risk: medium
status: verified
keywords: [fin_employee_registry, roster, last_beat, pure_interactive, 停滞误报, KPI假红, cockpit]
scope: [cockpit_face, flow_pulse, roster判据, 显示面]
summary: 活性判据全量套 status='active' 行——无心跳径席形（pure_interactive/COS）被当无头岗钉红，业主离场>10min必误报
first_seen: 2026-11-04
last_verified: 2026-11-04
evidence_refs:
  - type: server_fact
    source: postgres
    query: "SELECT name,status,meta->>'form',last_beat FROM fin_employee_registry WHERE name='COS'"
    observed_at: 2026-11-04T05:00:00Z
  - type: doc
    ref: cmd/core/cockpit_face.go:64
  - type: doc
    ref: docs/tasks/T-0344-名册卡COS停滞误报-纯交互席出心跳判据.md
---

## 事实/症状

业主驾驶舱「①存活」卡报「COS 停滞」。生产账实查：COS 行
`status='active'`、`meta.form='pure_interactive'`、last_beat 死寂——
账是对的，病在判据：roster worst-lag 查询 `WHERE status='active'` 全量
生效，未豁免席形。COS=纯交互秘书席（T-0322：无 daemon 无心跳径，
业主离开即闲是设计态），离场>10min 必然假红——owner_interruption
方向误报税（业主被假红叫回来）。

## 根因或策略解释

判据先于席形多样性落地：T-0308 钉 600s 阈时花名册只有会 Beat 的
无头岗；T-0322 新增 pure_interactive 席形后判据未回头审覆盖面。
类级形：**活性/健康判据对实体全集生效时，新形态实体（无该信号径的
形态）必然被钉病**——加新席形/新臂/新账形态时须回审所有全量判据。

## 处置

roster 查询加豁免谓词 `meta->>'form' IS DISTINCT FROM 'pure_interactive'`
（cockpit_face+flow_pulse 同口径同改——注释钦定勿双写防两端漂移）；
`IS DISTINCT FROM` 保 NULL form 行仍计（headless 岗零误豁免）。回归
锚 TestCockpitRosterPureInteractiveExempt 钉「PI 死寂豁免+无头岗死寂
仍触红」双向。

## 证据

- check: `go test ./cmd/core -run TestCockpitRosterPureInteractiveExempt`（O_TEST_PG_DSN 5434）
- result: PASS——PI 桩死寂 1h 豁免（worst≠PI、lag≤600），HD 桩死寂 worst=HD lag>600 触红
- ref: T-0344

## 反证条件

- check: 花名册出现第三种席形（如 semi_interactive 有心跳但稀疏）
- falsifies_when: 新席形心跳语义与 headless 不同但共享同 key 豁免失败——
  此时谓词白名单制须改 form→判据映射表，本条仅证「豁免席形」方向成立

## 适用边界

仅覆盖「有/无心跳径」二形态谱系；若未来出现心跳语义异形的席形
（稀疏心跳/事件代心跳），白名单谓词须升档为 form×判据映射。
600s 阈与 headless 覆盖面归 T-0308 管辖不在本条。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: 虚报
- query: n/a
- threshold: n/a（豁免由：判据本体=KPI 卡红即自监察面——修复后判据
  谓词内嵌豁免，复发监测=回归测试锚+票面验收「部署后生产亲验」随
  下一部署窗； gauges.tsv 表属生产数据面，本坑=显示判据码面，另立
  SQL 表测「豁免谓词是否还在码里」无确定判据可表达）
