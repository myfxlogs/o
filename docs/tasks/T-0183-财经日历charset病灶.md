# T-0183 财经日历 feed charset 病灶——windows-1252 解包恒败

## 态：可领

## 病灶（24h 实拍）

`[财经日历] feed 解包败: xml: encoding "windows-1252" declared but
Decoder.CharsetReader is nil`——24h 21 次恒败=**T-0070 高影响级
事件预推面全灭**（feed 每次拉到但解不出=高影响财经事件零预推）。

根因钦定：Go `encoding/xml` 对非 UTF-8 charset 需显式注册
`CharsetReader`——`golang.org/x/net/html/charset` 一行接入型
（`decoder.CharsetReader = charset.NewReaderLabel`）。

## 施工面

internal/platform/notify/calendar.go 的解包点接 CharsetReader
（x/net 依赖 go.mod 核——间接依赖可能已在）；桩=windows-1252 声明
xml 样本解出条目即过。

## 验收锚

1. 桩：windows-1252 声明样本→条目解出。
2. 部署后 24h 零 `财经日历.*解包败`。

### 决策注（派前钉——迁址裁决）

不迁 dev：日历件挂 o-core alerter/去重台账同进程面；迁需飞书
凭据面随件上 dev（凭据扩散面负收益）——微件原地修。

## 结果回填

- calendar.go 解包点接 CharsetReader：xml.NewDecoder+charset.
  NewReaderLabel（x/net v0.57.0 已在 go.mod——间接升直接引用面）；
  windows-1252 声明 charset 不再恒败。
- 桩锚 TestCharsetReaderWindows1252：windows-1252 声明样本→NFP 条目
  解出绿（T-0070 高影响预推面根修实证）。
