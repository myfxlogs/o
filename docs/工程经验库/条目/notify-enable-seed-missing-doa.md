---
schema_version: 2
id: notify-enable-seed-missing-doa
kind: pitfall
risk: high
status: verified
keywords: [app_config, 种键, notify.feishu.*.enabled, dead-on-arrival, fixture 真形分歧, 迁移漏种, 静默吞提醒, T-0382]
scope: [migrations, notify 推送面, app_config owner 类键, 施工验收]
summary: 功能件迁移建了表却漏种 enable 开关键——生产 app_config 无键→on()=false→命中走「推送关只翻位」径静默 fired，业主永收不到通知（核心语义破产+前端「已提醒」假象）；fixture on()=true 桩掩盖生产真形键缺=false 分歧——新通知件种键=迁移随行硬件+种键断言桩测钉死
first_seen: 2026-11-06
last_verified: 2026-11-06
evidence_refs:
  - type: commit
    ref: "返修裁决驳回件——fin_0119 漏种 notify.feishu.price_alert.enabled（DOA）；兄弟开关 fin_0100/0101/0102 同式种 'true'::jsonb 在案本件漏跟"
  - type: doc
    ref: "migrations/fin_0119_price_alert.sql（补种尾段）+ internal/platform/notify/price_alert_test.go（种键断言钉）"
  - type: doc
    ref: "docs/prompts/T-0382-返修1.md（裁决发现全文——DOA 定性+修径）"
---

## 事实/症状

T-0382 价格提醒全链交付且桩测四锚全绿，但功能 dead-on-arrival：
迁移建了 `fin_price_alert` 表却漏种开关键
`notify.feishu.price_alert.enabled`——生产 app_config 无此键→
`on()`=false→提醒命中走「推送关/凭据缺——不发只翻位」径，行翻
fired 而飞书永不发出；前端卡面还显「已提醒」假象。业主令
「飞书通知我」核心语义整体破产，且无任何报错/升件露头（静默）。

## 根因或策略解释

- **fixture↔生产真形分歧**：桩测以 `on()=true` 桩注入，断言的是
  「开关开时的行为」——键在生产侧是否被种无人核；测试面全真而
  部署面缺数据，DOA 型缺陷天然穿过桩测；
- **同式先例未跟**：兄弟开关 fin_0100(regime)/fin_0101(venue_req)/
  fin_0102(econ_digest) 全部由本件迁移自种 `'true'::jsonb`——新通知
  件未抄同式是种键漏落的直接因；
- **owner 类键无在任补救径**：notify.* 属业主面写域，DM 写径物理
  不可达——漏种后无法热补，只能迁移种。

## 处置

- fin_0119 尾部补种 `INSERT INTO app_config(key,value) VALUES
  ('notify.feishu.price_alert.enabled','true'::jsonb) ON CONFLICT
  (key) DO NOTHING`（同式字面——勿写成 '"true"' 串）；
- `TestPriceAlertFireOnce` 补种键断言：Migrate 后查
  `SELECT value::text FROM app_config WHERE key=...` 应='true'——
  钉死 fixture 与生产真形分歧（迁移自携种，键缺/值歪测试即红）；
- 施工自查新增一锚：**凡新通知/开关件=迁移随行种键+断言桩测**
  （兄弟件同式字面照抄，ON CONFLICT 幂等）。

## 证据

- check: `SELECT value::text FROM app_config WHERE
  key='notify.feishu.price_alert.enabled'`（生产补种前=零行）；
- result: 返修后='true'（jsonb 字面）；种键断言桩测 PASS（Migrate
  全链自携种）；
- ref: docs/prompts/T-0382-返修1.md（裁决发现全文）+本支返修收口
  commit。

## 反证条件

- check: 生产 `SELECT count(*) FROM ops_event WHERE event='价格提醒'
  AND note LIKE '%推送关%'`——补种部署后新 fired 行仍携「推送关」
  注=补种未生效或键名写岔；
- falsifies_when: 补种后首命中仍零飞书且 note 携「推送关」→结论
  「键缺是唯一因」失效（须转查 chatID/Sender 装配面）。

## 适用边界

- 一切 app_config enable 键门控的新通知/功能件（notify.\*/events.\*/
  digest.\* 族）——读侧 `on()=false` 静默降级件均中此坑谱系；
- 非 enable 类键（阈/参数有默认回退）不中——坑谱=「键缺=功能死」
  型 fail-closed 默认。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a（豁免由：复发闸已钉在迁移侧——种键断言桩测随
  check_all 全链跑，键缺/值歪=测试即红，确定性覆盖优于产线巡检；
  gauges.tsv 新表属产线数据面增编，超本返修票面窄域另立）
