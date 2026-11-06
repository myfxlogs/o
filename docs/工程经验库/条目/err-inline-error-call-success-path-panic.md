---
schema_version: 2
id: err-inline-error-call-success-path-panic
kind: pitfall
risk: high
status: verified
keywords: [nil error, Error() 裸调, 内联解引, panic, rows.Err, 成功径, Go, 502]
scope: [cmd/core, 全域 error 返回值消费面]
summary: 对 error 返回值裸调 .Error()（srows.Err().Error() 形）=成功径对 nil 接口做方法调用 panic——每调用必炸产线 502；先取后判是唯一合法形
first_seen: 2026-11-06
last_verified: 2026-11-06
skills: []
evidence_refs:
  - type: ticket
    ref: T-0390
  - type: server_fact
    ref: "生产 docker logs `http: panic serving …/quotes_face.go:112`（业主实屏「行情·价格提醒 该段数据缺 HTTP 502」）"
  - type: doc
    ref: cmd/core/quotes_face.go:112
  - type: test
    ref: cmd/core/quotes_face_test.go TestListPriceAlertsEmptyOK/TestListPriceAlertsRowsOK
---

# 对 error 返回值裸调 .Error()=成功径 panic（T-0390）

## 坑

`return res, connect.NewError(code, srows.Err().Error())`——`rows.Err()`
成功径=**nil error 接口**，对 nil 接口做方法调用=nil pointer panic。
败径永远走不到这行语义（Err() 非 nil 才有意义），于是该写法**只在
成功径炸**——每调用必炸，功能整体 DOA，桩测漏「成功径 Err()==nil
不炸」断言时完全无感过闸。

## 修

先取后判两步——`err` 返回值消费**永不内联方法调用**：

```go
if err := srows.Err(); err != nil {
    return nil, connect.NewError(connect.CodeInternal, err.Error())
}
return res, nil
```

## 防

- 全仓审计形：`\.Err\(\)\.Error\(\)` / `\)\.Error\(\)`——任何「函数
  返回值上直接链 .Error()」的写法都是嫌疑面（Commit/Rollback 尾部
  同理）；`if err != nil`/`return rows.Err()` 形合法（nil 是合法值）。
- 桩测必须钉**成功径**（零行+有行两态）断言「不 panic 且返 200」——
  只测败径的桩拦不住「成功才炸」型病灶。
- 与 typed-nil-interface-panic-loop 互补：彼坑=typed nil 装接口骗过
  `== nil` 判；本坑=对 nil 接口直接调方法——同为「接口 nil 语义」双生子。
