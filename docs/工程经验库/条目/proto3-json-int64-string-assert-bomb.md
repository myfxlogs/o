---
schema_version: 2
id: proto3-json-int64-string-assert-bomb
kind: pitfall
risk: low
status: verified
keywords: [proto3, JSON, int64, string, connectrpc, 桩测断言, panic]
scope: [go, 测试, proto, connectrpc]
summary: proto3 JSON 的 int64 编码是 string 非 number——桩测对 int64 字段做 float64 断言必 panic；走 strconv.ParseInt 解串
first_seen: 2026-11-07
last_seen: 2026-11-07
evidence_refs:
  - type: provenance
    ref: T-0442 TestAlertSoftDelete panic（cmd/core/quotes_face_test.go id 解串 strconv 化）
---

## proto3 JSON int64=string——桩测 float64 断言 panic 炸

- **实证**：T-0442 桩测从 `CreatePriceAlertResponse.rows[0].id`（int64）取值，
  `row["id"].(float64)` 直断言——connectRPC/protojson 对 64 位整数的 JSON 编码
  是**字符串**（`"1"` 非 `1`，JS number 精度守恒钦定）→interface 转换 panic
  崩测试进程。
- **教训**：proto3 JSON 形制的 64 位整数（int64/uint64/sint64/fixed64）一律编码
  为十进制字符串；HTTP 桩测断言取这类字段走 `strconv.ParseInt(s,10,64)`——
  double 断言只在 double 字段合法。int32 才是 JSON number。
- **收口**：id 解串 strconv 化后测试复绿（TestAlertSoftDelete PASS）。
