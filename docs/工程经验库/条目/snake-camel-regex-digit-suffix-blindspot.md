---
schema_version: 2
id: snake-camel-regex-digit-suffix-blindspot
kind: pitfall
risk: high
status: verified
keywords: [camelCase, snake_case, 数字尾缀, 双轨序列化, 手拼 JSON, Invalid Date]
scope: [frontend, proto, serialization]
summary: snake→camel 换算正则 _([a-z]) 不含数字——energy_apr_7d 类数字尾缀键静默落穿；同数据双轨两份序列化（手拼 REST+proto）人肉对拍键名必炸
first_seen: 2026-10-26
last_seen: 2026-10-26
evidence_refs:
  - type: provenance
    ref: T-0156（前端评估实证+本地渲染实锤）
---

## snake→camel 正则数字尾缀盲区 + 双轨序列化键名漂移

- **实证**：后端手拼 map 键 `energy_apr_7d`（trx_yield_face.go:48），前端 shim `camel()` 正则 `/_([a-z])/g` 不含数字——`energyApr_7d` ≠ proto 字段 `energyApr7d`，收益率卡 7d/30d 四格生产恒显「—」。同因族：`updated_at` RFC3339 string 直接喂 proto Timestamp 读取件（timestampDate）→「更新于 Invalid Date」。本地渲染复现实锤。
- **规则**：①凡写 snake→camel 递归换算，正则必须含数字段（`_([a-z0-9])`）；②根本病=同数据双轨两份序列化（手拼 REST JSON + proto Connect 同 mux）人肉对拍键名——收口为 proto 单源后此坑根除（T-0156 步1-4）；③string 时间戳喂 proto Timestamp 读取件前必须显式转换，类型断言 `as T` 会把形状错配压成运行期静默。
