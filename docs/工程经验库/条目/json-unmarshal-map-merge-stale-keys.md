---
schema_version: 2
id: json-unmarshal-map-merge-stale-keys
kind: pitfall
risk: medium
status: verified
keywords: [Go, encoding/json, Unmarshal, map, 复用, 残留键, 测试断言, 二次解形, 假绿]
scope: [test, cmd/core, JSON 断言面]
summary: json.Unmarshal 复用非空 map 不清底——二次解形小/空载荷残留首轮键，断言读到混合数据（轻则假红误诊，重则假绿——断言的是残留不是本轮回包）
first_seen: 2026-10-27
last_verified: 2026-10-27
skills: [KB维护]
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0174-readcred端点泛化venue束.md（结果回填·偏离测试坑）
  - type: code
    ref: cmd/core/readcred_test.go——TestReadcredMapEntitlementGates 二次解形复用 m，`{}` 回包残留首轮 okx 三件
repro_or_symptom: 同一 map 变量先解全量回包、翻数据后再解空/缩量回包——断言 len(m)==0 却见首轮键全在；handler 侧实测（cfg 快照/DB 行/allow 集）全对仍「复旧值」
root_cause: encoding/json 对非 nil map 目标按 JSON 键合并写入、绝不先清底——空对象 `{}` 零写入，旧键原地存活；测试作者把 Unmarshal 当「替换」语义用
fix_and_prevention: ①二次解形一律新变量承接（var fresh map[...]...）或解前 m=nil ②断言回包形状的测试禁跨数据态复用解形目标 ③假红信号：DB/快照实证已翻而断言见旧值=先查解形目标复用
notes: 假红方向（本轮）浪费一轮回诊；同坑反转即假绿——断言读到的是上一数据态，测试恒真——风险高于假红
---

## 现象

T-0174 readcred 白名单测试：首轮 `json.Unmarshal(rec.Body, &m)` 解出 okx 三件；翻白名单为空数组后再解 `{}` 回包进同一 `m`，断言 `len(m)==0` 却仍见 okx 全件——handler/DB/快照三面实证均已翻新。

## 根因或策略解释

`encoding/json` 对非 nil map 目标按 JSON 对象键合并写入，不清底：`{}` 不写任何键，首轮键全部原地存活。Unmarshal 是「合并进」不是「替换为」。

## 处置

测试二次解形改新变量（`var empty map[string]map[string]string`）承接后断言，测试转绿。

## 证据

- check: `go test ./cmd/core/ -run TestReadcredMapEntitlementGates`
- result: 复用 map=假红（残留 okx）；新 map=PASS
- ref: docs/tasks/T-0174-readcred端点泛化venue束.md

## 反证条件

- check: Go 标准库若改为 Unmarshal 前清空非 nil map 目标
- falsifies_when: encoding/json 文档/行为变更——非 nil map 解形先置空

## 适用边界

Go encoding/json 全版本（ 至今合并语义稳定）；泛化到任何「同目标二次 Unmarshal 小载荷」场景——结构体目标字段覆盖语义不同（零值字段不写），map 目标是残留重灾区。
