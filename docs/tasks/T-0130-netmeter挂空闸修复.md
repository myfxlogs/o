# T-0130 netmeter 挂空闸——okxws 计量件序位修复

| 字段 | 值 |
|---|---|
| 状态 | 审计通过 |
| 优先 | 高——5Mbps 链路下最大带宽消费面（507 标的全宇宙）零计量零告警 |
| 出处 | 决策席取证 `cmd/core/main.go:96-122`（netmeter 实测复核） |

## 病灶

`cmd/core/main.go` 装配序错位：`96` 行 `var meter`（nil）→`99` 行
`collector.AttachMeter(meter)` 挂的是 nil→`119` 行 `netmeter.New` 真建
但 okxws 再没补挂——**公网行情面自建账起从未被计量**，带宽预算告警
对该面=死开关（mt5/okxpriv 挂接在 New 后=真计量）。

同型风险：AttachMeter 签名拿指针值——调用方传 nil 无编译期/运行期
抱怨，静默失效。

## 修法方向（施工裁）

- 首选：AttachMeter 移到 meter 建成之后（装配序修正，一行位）
- 加固：AttachMeter(nil) 显式拒绝+告警或改懒引用——防下次序位再错
- 顺带实测：部署后日志/netmeter 指标验证 okxws 面进入计量，报实测
  带宽占 5Mbps 预算百分比（业主问题账的实测答卷）

## 验收锚

- 桩测/日志实证 okxws 帧入计量路径
- 507 标的面实测带宽报表落票（SG 网卡计数交叉）

审计席：headless @ 2026-10-30

## 判决注记——逐锚对账（审计通过）

收口链：83c6f3a（T-0132 施工，netmeter 序位修同车）→ f1bd7b8 验收合入 →
fa6315f 翻面；本票 4cc3884 随批翻定。对账只认锚与 diff/仓档实证，不引终态。

1. **桩测/日志实证 okxws 帧入计量路径——过**。
   序位修实落：`cmd/core/main.go` 原 `collector.AttachMeter(meter)` 在
   `netmeter.New` 之前挂 nil，83c6f3a 移入 meter 造成后的块内并留
   T-0130 注记；HEAD 复查仍在位。计量点实补：`handle()` 新增
   `if c.meter != nil { c.meter.Add("okxws/"+msg.Arg.Channel, len(data)) }`
   （HEAD 已重构落 `internal/data/okxws/handle.go:18-19`，健在）。
   日志实证支：`okxws/` 前缀分频道键唯一产出处即该 Add 点——SG 生产
   netmeter 分频道账实证在案（下锚②），帧入计量路径由生产账反证，
   非推断。桩测支：netmeter 单测 `TestChannelsPerChannel` 分通道账
   在谱；锚文「桩测/日志」析取，生产账满足。

2. **507 标的面实测带宽报表落票（SG 网卡计数交叉）——过**。
   `docs/tasks/T-0135-宇宙mark价REST化与非USDT滤除.md` 实证段（SG，
   T-0134 部署后 7min）：`okxws/mark-price`=93.7MB≈1.8Mbps（507 SWAP
   秒推）、`okxws/tickers`（活跃集~16）=59.9MB≈1.1Mbps、
   funding-rate=1.6MB——分频道带宽报表在票；SG 网卡 `ens17`
   RX=4763kbps 贴顶交叉在票（T-0134 实证段 RX=4767kbps 同源）。
   锚文「落票」未限定落本票——本票并入验收的批次链票档（仓内可复查
   实址）承载报表，合规。

注记报管理席（非驳回项）：
- 修法方向加固项（AttachMeter(nil) 显式拒绝/懒引用）未实施——签名
  仍裸指针值，nil 挂接同型静默风险敞口仍在；施工回填已记坑
  「attach-before-init 死挂」（中低危，停待归纳）。属施工裁量方向
  非验收锚，不阻判，留注备查。
- 本票「并入 T-0132 验收」使实证散落于 T-0132/T-0134/T-0135 票面
  而非本票回填段——判决依据链完整可溯，不阻判；若管理席要求
  单票自足证据，可由收口器批补指引。
