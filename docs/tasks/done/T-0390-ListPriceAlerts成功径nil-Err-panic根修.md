# T-0390 ListPriceAlerts 成功径 nil.Err() panic 根修

- 状态：完成（2026-11-06 决策席裁决通过——收口 7b3bb14c 合
  master d5253e8e；P0 即发窗部署 o-core:d5253e8e57f2 健康绿，
  新容器 panic 日志清零）
- 领取：devin-t0390 @ 2026-11-06（DM 常径机派；P0 生产功能 DOA）
- 优先级：**P0**——产线每调用即 panic，价格提醒功能全瘫
- 收口移交归：决策席

## 病灶（生产实证）

`cmd/core/quotes_face.go:112`：
`return res, connect.NewError(connect.CodeInternal, srows.Err().Error())`
——`srows.Err()` 成功径=nil，对 nil error 调 `.Error()`→nil pointer
panic。生产 docker logs 实证：`http: panic serving …/quotes_face.go:112`
每调用必炸→前端恒 502。桩测漏「成功径 Err()==nil 不炸」断言。

## 范围

1. quotes_face.go:112 修：先取 `err := srows.Err()`，非 nil 才
   NewError，nil 则 `return res, nil`；
2. 同仓审计：全仓 grep `.Err().Error()` 同型裸调——一并列清单修
   （模式=Err() 可能 nil 的 rows.Err/tx.Commit 尾部等）；
3. 桩测钉死：ListPriceAlerts 成功径断言——零提醒行/有行两态
   调用不 panic 且返 200/正常 payload（先红后绿：修前复现 panic
   断言必红——T-0388 纪律适用）。

## 验收锚

① 修前红桩：裸调形态复现 panic 断言红；
② 修后：ListPriceAlerts 实调返 rows+symbols 不炸；
③ 全仓 `.Err().Error()` 同型审计清单落回填；
④ check_all 绿戳对口；镜像面件——本件 P0，发窗不等攒批。

## 红线

- 只修 panic 与成功径语义——查询逻辑/前端零动；
- 不动真单径。

## 结果回填

### 实现摘要

- `cmd/core/quotes_face.go:112` 根修：`srows.Err().Error()` 裸调改为
  先取 `if err := srows.Err(); err != nil { return nil, connect.NewError(...) }`，
  nil 径 `return res, nil`——成功径不再对 nil error 调 `.Error()`。
- 新增 `cmd/core/quotes_face_test.go`：connecthttp 真挂载桩测两态
  （earn_face_test.go 同式 testutil 独立 schema+Migrate(fin)）。

### 测试证据（先红后绿）

- ① 修前红：`go test -run TestListPriceAlerts ./cmd/core/`——panic
  `runtime error: invalid memory address` 于 `quotes_face.go:112`，
  栈穿透 connecthttp→ServeHTTP→tRunner（生产 `http: panic serving` 同形）；
  `TestListPriceAlertsEmptyOK` 复现炸，断言红。
- ② 修后绿：`TestListPriceAlertsEmptyOK`（零提醒行：200+symbols 枚举
  非空+rows 空）、`TestListPriceAlertsRowsOK`（有行：200+rows×1
  symbol/state/hasQuote/last 投影对+symbols×2）双 PASS；
  `go test ./cmd/core/` 全包绿 85.3s。
- ③ 同型审计清单（全仓 grep，排除 docs/票面自身引用）：
  - `\.Err\(\)\.Error\(\)` 裸调：**1 处**——quotes_face.go:112（本修，已消除）；
  - `\)\.Error\(\)` 任意内联解引：修后 **0 处**；
  - `rows.Err()`/`ar.Err()`/`ctx.Err()` 其余 60+ 用点：全部为先取后判
    或直接 return（nil 合法值），零裸解引；
  - `Commit()/Rollback()` 内联 `.Error()`：**0 处**；
  - `GetError().GetMessage()`（tools/mt5verify）：proto getter 零值安全，
    非 error 接口解引——非同型。
- ④ check_all：全链绿——build✓ vet✓ import✓ gofmt✓ env✓ rejects✓
  config-write-domain✓ migrations✓ bridge-audit✓ tolerate-report✓
  go test 全谱✓（含 cmd/core 382.7s 全包）——戳落
  check-green-<收口SHA>（提交后重跑对口）。

### 偏离说明

- 无越单——只动票面钦定两面：panic 行修复+新增桩测；查询逻辑/前端/
  真单径零触碰；diff 面=quotes_face.go、quotes_face_test.go、本票、
  KB 条目+索引（D-585 坑处置同 commit 径）。
- 坑处置：坑成立——`docs/工程经验库/条目/err-inline-error-call-success-path-panic.md`
  同 commit 入库（kbgen 索引已重建；与 typed-nil-interface-panic-loop
  互注双生子）。
