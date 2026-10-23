# T-0018 app_config 标量契约根修（线上实证 bug——研究面被自家闸锁死）

| 字段 | 值 |
|---|---|
| 状态 | 施工中 |
| 提出 | `[角色:决策]` 2026-10-23 |
| 类别 | 缺陷修复（线上实证）+契约钉死 |
| 依赖 | 无 |
| 领取 | devin-t0002 @ 2026-10-23 |

## 背景

开发机常驻实证：`[research] scan` 每 60s fail-closed 于 `research.funding.taker_fee_bps: json: cannot unmarshal number into Go value of type string`——迁移种子写 jsonb **原生数字**（`5.0`/`0.05`），而 research/risk 消费方 `var s string; json.Unmarshal(raw,&s)` 按**字符串编码**读。闸工作完全正确（拒出卡不产垃圾），但公司被锁死在「永远评估不了」态——上线即实证。

## 裁决（契约钉死，禁再议）

**app_config 标量约定=JSON 原生类型**：数字存数字、布尔存布尔、字符串存字符串。**禁字符串编码**（`"5.0"` 式双编码=nox env-string 残留病灶）。种子写数字是对的，消费方 `var s string` 是错方。

## 范围

1. **platform config client 加类型化读器**：`GetFloat(k)`/`GetBool(k)`/`GetString(k)`/`GetInt(k)`——单点实现，接受原生 jsonb 类型；类型不符=error（fail-closed 保留，但「数字存字符串」这种双编码不再被定义）。可选宽容：数字接受 `json.Number` 与 numeric string 两形（历史种子可能有字符串残留——读器宽容、写约定严格）。
2. **消费方全改走读器**：research/funding、risk、lifecycle、execution、ledger、guard 域内所有 `var s string; Unmarshal; ParseFloat/=="true"` 段全改 `cfg.GetFloat/GetBool`——grep 全清，不许留散件。
3. **回归锚**：dev 库原样数据下 `[research] scan` 出评估卡（fin_opp_eval 行）或明确判 unevaluable（有 verdict 行即通——零行才是病）；risk getB 同款读原生布尔。
4. **坑入经验库**（D-585 修复票必填）：种子契约与消费方双编码漂移——「schema 定了 jsonb 但没定值编码约定」是根因层。

## 验收标准

- [ ] config client 四读器入库；域内 `ParseFloat/var s string` 消费段 grep 清零（含 risk/execution/lifecycle 全谱——不止 research）
- [ ] 裸数字种子+原生布尔种子下各域读通；类型不符 fail-closed 测试锚
- [ ] dev 库实证：research scan 每拍不再报 unmarshal 错；fin_opp_eval 产出 verdict 行（proposed/rejected/unevaluable 皆可，零产出=病未愈）
- [ ] 经验库条目同票入库+索引重建
- [ ] 机检全绿；CI 绿

## 范围外

- 私流守凭据播种（运维动作，见备注）、新员工上岗（镜像 rebuild 即到位，随 T-0015 合入后一并）
