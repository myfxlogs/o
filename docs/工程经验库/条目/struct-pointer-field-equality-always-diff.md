---
schema_version: 2
id: struct-pointer-field-equality-always-diff
kind: pitfall
risk: low
status: verified
keywords: [结构体判等, 指针列, h!=s, reflect.DeepEqual, 全表diff, 反向桥, 恒覆写]
scope: [projector, reverse, golang, diff]
summary: 含 *string/*int64/*time.Time 指针列的行结构体用 `h != s` 判等=按指针地址比——同值不同指针恒不等，全表 diff 退化为「每拍全表覆写」（结果对但浪费）；判等须按所指值：reflect.DeepEqual 或逐字段解引用比
first_seen: 2026-11-04
last_seen: 2026-11-04
evidence_refs:
  - type: doc
    ref: internal/projector/reverse.go:strategyRow（proposal/notional/crit 指针列，h!=s 恒不等先例）
  - type: doc
    ref: internal/projector/reverse_fin.go:revTableDiff（reflect.DeepEqual 判等修正式）
---

# 指针列结构体 `!=` 判等=恒不等→diff 恒覆写（T-0326 实证）

## 现象

双侧全表 diff 惯用形：行结构体入 map，`h != s` 判缺/差行进 upsert 批。
结构体含可空列的 `*string`/`*int64`/`*time.Time` 字段时，`!=` 按**指针
地址**比——两侧各 Scan 出的同值行是不同指针→恒不等→**每拍全表行进
覆写批**。结果幂等故功能不破，但隧道载荷=全表量而非差行量（「仅差行
过隧道」设计前提破），表涨后浪费线性放大。

## 修式

- 判等改 `reflect.DeepEqual(h, s)`——指针按所指值递归比（time.Time 同
  源扫描表示一致即等）。
- 或逐字段解引用比（`*string`→值等、`*time.Time`→`.Equal()`——
  repairProposalStates 的 reasonEq/timeEq 先例）。

## 判据

写行结构体含指针列的 map diff 时：**禁 `!=`/`==` 直比**——要么
DeepEqual，要么字段级比较函。存量 `strategyRow`（reverse.go）同病灶
在案（proposal/notional/crit 指针列——fin_strategy 每对账拍全表覆写，
行量百级故未爆），修式=同 DeepEqual 化（本件新码已修，存量顺手改
与否归后续票裁）。
