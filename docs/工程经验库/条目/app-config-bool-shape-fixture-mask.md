---
schema_version: 2
id: app-config-bool-shape-fixture-mask
kind: pitfall
risk: high
status: verified
keywords: [app_config, jsonb, 布尔形, kill_switch, 测试夹具, 异形种子, 静默失效, dd硬线]
scope: [app_config 消费方, internal/guard, 一切手写 json.Unmarshal 读径, 测试夹具种子]
summary: 产线种子=jsonb 布尔而消费方手写 string-only unmarshal——测试夹具全键 to_jsonb(text) 种字符串形把异形病灶遮蔽，安全硬线全程静默未判
first_seen: 2026-11-16
last_verified: 2026-11-16
skills: []
evidence_refs:
  - type: ticket
    ref: 工单#20799 dd-hardline-dead
  - type: code
    ref: internal/guard/ddline.go killSwitchOn
  - type: migration
    ref: migrations/fin_0008_risk.sql ('risk.kill_switch','false')=jsonb 布尔
---

# app-config-bool-shape-fixture-mask —— 产线布尔形 vs 夹具字符串形：异形被测试遮蔽，硬线静默未判

## 坑

`app_config.value` 是 jsonb，值可为布尔/数字/字符串多形。`fin_0008` 种子
`('risk.kill_switch','false')` 落 **jsonb 布尔 false**；而
`guard/ddline.go killSwitchOn` 手写 `json.Unmarshal(raw,&s)` 只吃字符串形
——产线恒 unmarshal 败→`dd_missing_param` 计数→`ddScan` 采点后即返，
**回撤硬线（-3%/30d→全平+kill_switch，业主令）全程未在判撤且零日志**。

遮蔽机制更毒：测试夹具 `to_jsonb($2::text)` 给**所有**键种字符串形，
与产线布尔形不同——单测全绿而产线病。数值键同族坑已有
`config-json-type-fragility`/`app-config-scalar-encoding-drift` 两案；
本案新增两条增量：①布尔形同样会踩；②**夹具形与产线种子形不一致=
回归网自带盲区**。

## 实证

工单#20799（2026-11-16 外席全局审查钱面取证）：30d 峰值 12853.92 后
4 个采点回撤≥3%（-11.0%/-11.0%/-80.8% 等，均系演习批/划转在途瞬时读数）
均未触线，kill_switch 仍 false；`updated_at=10-27` 未再变=闸面从未动作。

## 修法（本件落地）

- 读径一律走平台双型读器 `config.Client.GetBool`（bool|"true"/"false"
  两形皆收）——与 `risk.Gate getB` 同口径；**禁再手写单型 unmarshal**。
- 测试夹具按产线种子形播种：`risk.kill_switch` 单列 `'false'::jsonb`，
  不再混入 to_jsonb(text) 字符串形批量键。
- 读败/异形须有可见出口：ddScan 读败支挂 `w.note` 限频日志（原静默）。

## 反证条件

- 新增/改动 app_config 消费方：先 `jsonb_typeof` 普查该键产线形，
  再对读径做双型核。
- 测试夹具种子：凡与迁移种子同键者，**形必须对齐产线**（bool 键种
  `::jsonb` 布尔，禁 to_jsonb(text) 一把抓）。
- 手写 `json.Unmarshal` 进标量前，先问平台 typed.go 是否已有双型读器。
