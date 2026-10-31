# T-0183 财经日历 feed charset 病灶——windows-1252 解包恒败

## 态：审计通过

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

## 审计判决

审计席：headless @ 2026-10-30

逐锚对账（收口 diff=6cbcfd5 全量亲核，merge=08ed695，fix 在
origin/master 祖先链实证）：

1. 桩锚（windows-1252 声明样本→条目解出）——达，三证齐：
   宣称件在册 internal/platform/notify/calendar_charset_test.go
   （TestCharsetReaderWindows1252，ffWeekly/ffEvent 字段与生产结构
   对拍一致）；审计席实跑 `go test ./internal/platform/notify
   -run TestCharsetReaderWindows1252` → 1 passed；生产码实态
   calendar.go:83-85 = xml.NewDecoder+CharsetReader=charset.
   NewReaderLabel，钦定修法逐字落实，解包败 log 前缀原样保留
   （锚② grep 键不破）。
2. 部署后 24h 零解包败——达，SG 生产面亲证（SSH 只读径）：
   实证窗 ≈43h 超锚——SG 首个 post-fix 镜像 ab62c2f952af 建于
   10-28 14:28Z，现役 o-core:0dbeb3d5d6ef（=origin/master HEAD，
   含 fix）healthy 在跑。
   端到端正实证=SG fin md_calendar_push 10-28 16:59Z→10-30
   09:47Z 连续 15+ 行（FOMC/BOC/ECB/GDP/PCE/BOJ）——行写入以
   解码成功为先决（Decode 败则 return nil 先出事件环），真实
   windows-1252 feed 被持续解出推送；零反证=SG 全部在存容器
   json 日志 grep 解包败 0 命中，现役容器 09:31Z 起生命内
   09:47Z 推送落地=现役实例解码实证。
   取证边界注记：重启前容器日志随容器删除不可得，部署后最初
   ~2.5h 无直接日志 grep 面——由 md_calendar_push 自部署当口
   起的连续落行覆盖式旁证（解码若败=零推送行，与落行矛盾），
   锚按达计。

非锚注记（报管理席酌处，不阻塞判决）：
- go.mod `x/net v0.61.0 // indirect` 标记未随直接引用转正
  （go mod tidy 应入直接 require 块）——回填「间接升直接引用面」
  文面与 require 块标记不符，外观疵不挡构建；x/ 族顺带
  v0.57→v0.61 升级非必需但自洽无害。
- 票态文「merge 6cbcfd5」实指施工 commit，merge commit=08ed695——
  文面微疵。
- 收口 commit 标「坑处置:无坑」——windows-1252 恒败属真坑
  （24h 高影响面全灭），未随票登工程经验库条目属可议项，
  报管理席酌是否补条目。
