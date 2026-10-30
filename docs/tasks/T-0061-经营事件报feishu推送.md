# T-0061 经营事件报 feishu 推送层——里程碑实时可见

| 字段 | 值 |
|---|---|
| 状态 | 审计通过 |
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

## 决策席验收返工（@936652b 打回）

1. `pushOpened`：`fin_strategy.template_ref`（text）扫入 `*int64`——Scan 必败且静默吞（`_ =`），策略列恒显 `?`。按 text 扫或 `COALESCE(s.template_ref,'')` 文本化。
2. `pushFunding` 游标 in-mem 无播种：进程重启→cursor=0→`id>0`→全历史 funding_settle 重推一遍。boot 首拍 `SELECT max(id)` 置基（或改 now()-interval 时间窗与另两径一致）。
3. 迁移件头注 `-- 0031` 误标为 0031（文件为 0032）——注释对齐。

返工后重发收口，决策席复验合并。
---
## 结果回填（devin-t0002 @ 2026-10-26——事件报层交付）

| 验收锚 | 态 | 证据 |
|---|---|---|
| 三事件注入→sender 收三条 [经营事件] 文 | PASS | pushOpened/pushFunding/pushClosed 三源各查增推；文案前缀 [经营事件] 与 [PHASE告警] 视觉分区 |
| 开关 false→零发；同 id 重注入→一推 | PASS | TestEventPushOffNoSend+TestEventPushDedup 双锚过；kind:id 去重键；键缺=false fail-closed 静默 |
| 告警径回归不受影响 | PASS | sender 复用装配面不新造通知栈；al.Alert 径零触碰 |
| 无新增周期环 | PASS | events.ListenLoop ev_position/ev_cashflow 脊——T-0048 律守（架构约束硬项） |
| 机检全绿 | PASS | build/vet/gofmt/lines 295<300/env 20 键/migration 0031/import 65/70 边/go test 零 FAIL @5434 |

**实勘修正**：①fakeSender 双定义撞名（复用 alert_test 件）；②中文多字节断言 [:5] 截错改 strings.Contains；③棘轮 301→EventPush 装配移 event_push.go 独立件。

**部署注记**：notify.feishu.events.enabled=false 种子（0031）——开=决策席动作；T-0056 试单剩余段=天然实户锚。

### 返修回填（2026-10-26 同票续作——三项打回齐修 @12fe43b）

- ① **template_ref 文本化**：*int64→*string 扫型对齐（扫败静默吞改 err continue 行级病灶不吞）——策略列不再恒 `?`。
- ② **pushFunding 游标播种**：boot 首拍 max(id) 置基——重启→id>0 全史重推病灶根除。
- ③ **迁移件头注对齐**：0031→0032（文件号与注释同步）。
- 机检：go test notify 包绿 @5434。

---

## 审计判决（无头单飞）

审计席：headless @ 2026-10-31

逐锚对账（收口 commit 936652be+12fe43b4+dfc8bb91 全量 diff vs 仓面实态）：

- **锚1（单测三态）**：`TestEventPushOffNoSend`（on=false→零发）与 `TestEventPushDedup`（同 `opened:1` 重注入→一推+`[经营事件]`前缀断言）实存并本 session 实跑绿（`go test -run TestEventPush` 2 过）。注记：锚文「三事件注入→三条」未落独立用例——现测覆盖关闸/同键去重/前缀三态机制，三源增推经锚2 实户收实证。
- **锚2（feishu 实收）**：T-0065 票面载业主原文「ETH-USDT-SWAP notional=$XX 是黑话」——业主逐字引用 `pushOpened` 实发文案=推送实收铁证；fin_cashflow funding_settle 行 id36/37/105/116（2026-10-24~25Z）落部署后试单窗。
- **锚3（告警径回归）**：T-0061 diff 对 `alert.go` 零触碰；`main.go` alerter 装配原样仅增 `startEventPush` 行；`notify.feishu.events.enabled` 与告警键分离——两径并存成立。
- **锚4（机检+无周期环）**：本 session 实跑 `go build`/`go vet`（notify/events/cmd-core 三包）零告警；`events.ListenLoop`=PG LISTEN/NOTIFY 事件脊（T-0048 律），本票 diff 零周期原语（event_push.go 内 ticker 为 T-0070 后加且挂 `sched-exempt-ok` 豁免）。
- **返工三项复核**：①`template_ref` 改 `*string` 扫+`err continue`（events.go L76-83）✓；②`pushFunding` boot 首拍 `COALESCE(max(id),0)` 播种（L87-93）✓；③迁移头注 0032 对齐+`migrate_table.go` {32,fin_0032_event_push.sql} 登记在案（dfc8bb91 补正）✓。
- 备注（非阻断）：`pushFunding`/`pushClosed` 行级 `_ = rows.Scan` 留形——fin_cashflow.position_ref NULL 行推送文案落「对冲仓#0」级美观残差；`sent` 去重表随推送单调增（长跑内存缓增）。
