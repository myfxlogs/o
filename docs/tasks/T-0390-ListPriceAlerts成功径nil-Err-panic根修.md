# T-0390 ListPriceAlerts 成功径 nil.Err() panic 根修

- 状态：已批（2026-11-06 业主实屏报——「行情·价格提醒 该段数据缺
  （读败：[unavailable] HTTP 502）」；COS 生产取证实锤）
- 领取：常径待派（DM 常径机派；P0 生产功能 DOA）
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

（施工写：实现摘要 / 测试证据 / 偏离说明）
