# T-0061 经营事件报 feishu 推送层——里程碑实时可见

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 优先级 | 中 |
| 涉及面 | `internal/platform/notify` `cmd/core`（新 worker 或挂信使）、`app_config` 键 |
| 证据日 | 2026-10-24（首仓序试单期里程碑业主侧不可见实证） |

## 病灶

推送面只有两档：**介入级告警**（rollback_failed/close_gave_up 等——出事才响）与**期界
报表**（日报节拍）。首仓序三里程碑（开仓确认/funding 结算落账/平仓+对账）业主侧完全
静默——真钱闭环首次跑通这件事本身即经营大事，靠人查库不合理。

## 目标

新增**经营事件报**档（event push，非告警语义）——里程碑事件实时推 feishu：
- `position_opened`：fin_position 新开仓行（venue/inst/notional/策略ref）
- `funding_settled`：fin_cashflow kind=funding_settle 落行（金额/仓ref）
- `position_closed`：fin_position 平仓行（含 settled_amount 盈亏）
- 三件起底；可扩：live 态晋升（fin_strategy_transition→live）、大额停泊/赎回（>阈）

## 设计钉

- **事件驱动**：挂 `ev_position`/`ev_cashflow` NOTIFY（禁新增周期环——T-0048 律）。
- **独立开关**：`notify.feishu.events.enabled`（与告警开关分离——业主可只关事件报
  不关告警）。键缺=false fail-closed 静默（不炸推送面）。
- **去重频控**：同事件一键一推（position_id+kind 去重）；风暴期同 kind 聚合窗
  （防批量行刷信道）。
- **与告警分径**：文案前缀 `[经营事件]` 与 `[PHASE告警]` 视觉分区；失败不升告警
  （推送败=rej 计数+log，不递归告警）。
- **零秘面**：只推金额/状态/时戳，禁凭据/密钥/内部 ID 细节超需。
- 复用现有 Alerter/sender 装配面，不新造通知栈。

## 验收锚

1. 单测：三事件注入→sender 收三条 `[经营事件]` 文；开关 false→零发；同 id 重注入→一推。
2. 部署后 T-0056 试单剩余段（开仓/结算/平仓）在 feishu 实收——本票的天然实户锚。
3. 告警径回归：原 al.Alert 推送不受影响（介入级与事件级两信道并存）。
4. build/vet/test 绿；无新增周期环（架构约束硬项）。
