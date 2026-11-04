# T-0316 scout metrics.go vet copylocks 根修——master 闸红追修件

| 字段 | 值 |
|---|---|
| 状态 | **施工中**（devin-t0316 @ 2026-11-04） |
| 立项 | `[角色:决策]` devin @ 2026-11-04（T-0315 裁决门禁对账实证） |
| 领取 | （空=未领） |
| 优先级 | 高——`go vet` master 恒红=check_all 全谱闸阻一切后续票 |
| 涉及面 | `internal/research/scout/metrics.go`（单文件；修法施工裁——索引迭代或 `[]*Board` 上游改型均可，以不拷锁为准） |

## 背景（裁决实证，勿再诊）

T-0315 裁决跑 check_all 实证：`go vet ./...` 于
`internal/research/scout/metrics.go:224` 报

```
range var b copies lock: o/internal/research/scout.Board contains
google.golang.org/protobuf/runtime/protoimpl.MessageState contains sync.Mutex
```

`for _, b := range boards` 逐值拷贝含锁 Board——86e1d47f（T-0313 合入）
引入，master HEAD（2cc88509）同诊在案，与 T-0315 diff 无关故不罚该件，
立本票追修。

## 验收标准

- [ ] `go vet ./...` 全绿（metrics.go:224 copylocks 诊灭）。
- [ ] `scripts/check_all.sh` 全谱绿盖戳。
- [ ] 同文件其余 range/赋值面无第二处拷锁（grep 级排查回填）。

## 范围外

- Board proto 型零改（含 MessageState 是 T-0312 第二刀钦定共享型——
  修迭代姿势不修型本身）。
- 不动 scout 业务逻辑/阈值。

## 施工回填（devin-t0316 @ 2026-11-04，分支 t/0316-vet-copylocks）

- 修法择：索引迭代 `for i := range boards { b:=&boards[i] }`——型本身零改
  （Board=proto 共享型含 MessageState 钦定件不修型）。
- `go vet ./...` 全绿（metrics.go:224 copylocks 诊灭）；同文件 grep 级
  排查其余 range/赋值面无第二处拷锁（scout_test.go:228 已索引迭代在案；
  venue_req.go `b:=m.Boards[best]` 指针型拷不触锁）。
- check_all 全谱：除 bridge-audit 末列 arch 水印对拍红=**生产态闸**（远程
  DB arch×源对拍非码面——本 diff 零涉；env 态红如实记档不归本票修）。
