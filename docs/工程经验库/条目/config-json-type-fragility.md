---
schema_version: 2
id: config-json-type-fragility
kind: pitfall
risk: high
status: verified
keywords: [app_config, jsonb, 型别敏感, getF, 静默失效, fail-closed倒错向]
scope: [config 热启, 判据参数, 一切 jsonb 标量读径]
summary: app_config 值型别敏感——getF 只吃 JSON 字符串形，裸数字静默废触发器（安全闸被配置型别哑掉零日志）
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: []
---
# config-json-type-fragility —— app_config 值型别敏感：getF 只吃 JSON 字符串，裸数字静默废触发器

## 坑
`app_config` 值是 jsonb——`getF` 实现 `json.Unmarshal(raw, &s)` 进 string 再 ParseFloat，**只认 `"20"` 字符串形，裸 `20` 数字即 unmarshal 败→missing_param→该判据整链静默跳过**。安全触发器被配置型别哑掉且零日志——fail-closed 倒向错方向（该响的触发器不响）。

## 实证
PHASE 仓 1（2026-10-24）：`exec.hedge_band_bps` 入库为裸数字 20（其余键均字符串形——单点异类），delta_drift 判据每 30s 扫一拍每拍 missing_param，30+ 分钟零触发零日志。改值为 `"20"` 字符串后 40 秒内真触发。期间 evaluate 的 5 个 missing_param 静默位全程黑盒。

## 反证条件
- 任何读 app_config 数值的径：读器必须双型兼容（json.RawMessage→先试 float64 再试 string→ParseFloat），禁单型假设
- 新增配置键：写值型别必须与读器约定一致——入库前 `jsonb_typeof` 核拍
- 静默拒类（missing_param/facts_missing/leg_anomaly）计数器必须有**可见出口**——计数增但零日志=黑洞，至少 debug 日志或周期摘要

## 机械化
- T-0046：getF 双型兼容+存量键型别审计脚本（jsonb_typeof 普查）；evaluate 各 skip 分支加结构化日志（限频）。
