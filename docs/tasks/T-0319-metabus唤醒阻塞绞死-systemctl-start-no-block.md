# T-0319 metabus 唤醒阻塞绞死——systemctl start 须 --no-block

| 字段 | 值 |
|---|---|
| 状态 | **施工中**（devin @ 2026-11-04） |
| 立项 | `[角色:决策]` devin @ 2026-11-04（T-0318 裁决期帧断流实证） |
| 领取 | （空=未领） |
| 优先级 | 高——事件链唤醒径静默绞死=席睡死无人知 |
| 涉及面 | `cmd/metabus/main.go` 唤醒调用点（一处） |

## 背景（实证在案，勿再诊）

2026-11-04 09:39:48 帧 max=1475 唤起 o-decision-seat（T-0318 裁决
run）；09:50 新 dm 帧到达时 metabus 调 `systemctl --user start
o-decision-seat.service`——unit 正 `activating`，start 同步排队等
activation 完成→**整个帧接收循环阻塞**：

- 对端 `ss` 实证：`CLOSE-WAIT` 1986B 缓冲堆积（后续心跳+帧全堆着无人取）
- 45s 看门狗同循环饿死——不触判死不重连，静默绞死与流死同症不同因
- 重启解绞后 resume 补帧 `max=1479` 即恢复（水位机制无恙）

区别于已修的「TCP CLOSE-WAIT 真死」：那次 Receive 不返/无缓冲堆积；
这次是唤醒调用阻塞致循环停摆，看门狗失效。

## 修法（施工裁，二选一或等价）

- `systemctl --user start --no-block <unit>`——job 入队即返
- 唤醒改异步 spawn（`exec.Command(...).Start()` 不等退出）

唤醒语义是「有活即醒」非「等它干完」——阻塞语义本就错配。

## 验收标准

- [ ] unit `activating` 态下灌新 dm 帧，帧循环不阻塞（日志连续+缓冲零堆积实证）。
- [ ] `go vet ./cmd/metabus` 绿；既有 hub/metabus 测试不破。
- [ ] 回填：阻塞根因+修法一行注。

## 范围外

- 不动 eventbus/hub/proto（服务端无恙）。
- 不重构唤醒映射表。

## 施工回填（devin-t0319 @ 2026-11-04，分支 t/0323-319-wake-relay）

- 修法择：`--no-block`（钦定二选一甲——job 入队即返；activating 态
  同步排队绞死帧循环根修；唤醒=有活即醒非等干完）。
- `go vet ./cmd/metabus` 绿；build 绿。
- activating 态灌帧实证=部署后实证件。
