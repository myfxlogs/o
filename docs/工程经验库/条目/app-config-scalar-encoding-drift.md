---
schema_version: 2
id: app-config-scalar-encoding-drift
kind: pitfall
risk: high
status: verified
keywords: [app_config, jsonb, 双编码, unmarshal, fail-closed, 标量契约, 种子消费漂移]
scope: [internal/platform/config, 全域 config 消费方, migrations 种子]
summary: schema 定了 jsonb 没定值编码约定——种子写原生数字而消费方按字符串解码，fail-closed 闸正确工作却把公司锁死「永远评估不了」态
first_seen: 2026-10-23
last_verified: 2026-10-23
skills: []
evidence_refs:
  - type: doc
    ref: internal/platform/config/typed.go
  - type: server_fact
    ref: dev 实证 [research] scan 每 60s fail-closed「cannot unmarshal number into string」
---

# app_config 标量双编码漂移（T-0018）

## 病灶

- 契约缺位：schema 定了 `value jsonb` 但**没定值编码约定**。
- 两写两读漂移：迁移种子写原生 jsonb 数字（`5.0`/`0.05`），消费方
  `var s string; Unmarshal(raw,&s); ParseFloat(s)` 按字符串编码读——nox env-string 残留。
- 症状：闸 fail-closed 正确工作但公司锁死在「永远评估不了」——
  **fail-closed 正确 ≠ 系统健康**，拒出卡不产垃圾同时也不产评估。

## 裁决（契约钉死，禁再议）

- **app_config 标量=JSON 原生类型**：数字存数字、布尔存布尔、字符串存字符串。
  禁 `"5.0"` 式双编码（写入侧）。读器宽容（历史字符串残留两形皆收）、
  写约定严格。
- 单点读器：`cfg.GetFloat/GetBool/GetString/GetInt`——消费方禁自写
  `Unmarshal+ParseFloat` 散件（grep 清零是门禁面）。

## 判据

- 新 config 消费一律走 typed readers；新种子写原生 JSON 字面量
  （`'500'`→number、`'false'`→bool、`'"x"'`→string——jsonb 解析语义）。
- 排查「闸一直拒」时先核：是判据真不满足，还是编码/读面把「能满足」误拒——
  闸正确工作下的系统性零产出是编码病灶体征。
