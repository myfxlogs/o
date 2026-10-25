# T-0130 netmeter 挂空闸——okxws 计量件序位修复

| 字段 | 值 |
|---|---|
| 状态 | 待派 |
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
