# T-0220 TRX 投票引擎 feishu 推送挂接——领奖/改投/异常/租金四事件上线

## 态：作废（superseded——11-01 清账定性·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**作废**——superseded：TRX投票feishu挂接交付并master（t/0220-trxvote-feishu merge在案）——验收债归T-0249
## 立项：业主令 @ 2026-10-28（原指令携 T-0210 撞号退回——施工席勘查报告在案，补立 T-0217）
## 域：后端 / TRX 自营盘自动化通知
## 依赖：T-0192 投票引擎启驶（已投产，vote_enabled=true）

## 病灶

TRX 业务四类事件（自动领奖/自动改投/引擎异常/能量租金日入）**零 feishu 推送**——引擎替业主赚钱或出异常，业主只能手动开 TRX 页看。trxvote 执行器已有 `Alert()` 调用（alerts.go 模板雏形齐备）但 **Alerter 未挂接**（只落日志）。

## 施工件

1. **挂接 feishu Alerter**：`cmd/core/trxvote.go` trxvoteArm 内注入（与 `earnW.AttachAlerter(alerter)` 同款，cmd/core/main.go:149 先例）。
2. **四类事件推送**（复用/扩形 alerts.go 模板）：
   - 自动领奖成功：`[投票领奖] +46.74 TRX (≈$15.33) tx=abc…`（USD 折算用 GetTrxYieldCurve currentPrice）
   - 自动改投执行：`[投票改投] SR-A→SR-B (年化差+0.12pp) 107.3k votes`
   - 引擎异常/停驶：`[投票引擎异常] 详情`（边沿触发不重复轰炸）
   - 能量租金日入 ≥10 TRX/笔：`[能量租金] +28.34 TRX`
3. **不推**：CMC 隔夜息（$1-2/天太碎）、普投心跳、微付款、CMC WS 心跳。

## 不动

- 前端零改动（feishu 是推送非页面）
- 改投参数不放宽（vote_max_rebal_per_day/min_interval 不动）
- 引擎配置不翻（vote_enabled 已 true 不动）

## 验收锚

- [ ] 18:00Z 窗领奖 → 飞书收到 `[投票领奖]` 推送
- [ ] 引擎异常测试注入 → 飞书收到告警
- [ ] 正常静默期 → 零推送（无噪音）
- [ ] 全量 go test 绿+四门禁过

## 施工序

worktree t/0217 + 全门禁 + 绿戳 → 推 master → 部署 → 18:00Z 窗验收 → 翻票销号。
