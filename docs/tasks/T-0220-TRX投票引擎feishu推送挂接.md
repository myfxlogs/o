# T-0220 TRX 投票引擎 feishu 推送挂接——领奖/改投/异常/租金四事件上线

## 态：待验（施工：zcode-t0220 @ 2026-10-28T11:05Z——分支 t/0220-trxvote-feishu）
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
（施工注：t/0217 为撞号让位残文——按业主钦定走 t/0220。）

## 结果回填

施工：zcode-t0220 @ 2026-10-28，分支 `t/0220-trxvote-feishu`（基 origin/master d0ba19b）。

### 实现摘要

- **Alerter 挂接**（施工件①）：`trxWiring` 增 `al *notify.Alerter`，`trxArms` 加参构造期注入（免竞态），cmd/core/main.go:229 传 alerter——与 earnW.AttachAlerter（main.go:149）同通道同款。
- **四类事件**（施工件②）：
  - 领奖：executor sink 双路（log+feishu），kind=`trxvote_claim`；`formatClaimAlert` 扩形 `[投票领奖] +46.74 TRX (≈$15.33) tx=…`——USD 折算价源 `LastCMCPriceMidWithTime("TRXUSDT")` 与 GetTrxYieldCurve currentPrice 同源（`Scheduler.SetPriceFn` 注入；无报价省略折算段不阻断推送）。
  - 改投：kind=`trxvote_rebalance`；`formatRebalanceAlert` 重写 `[投票改投] SR-A→SR-B (年化差+0.12pp) 107.3k votes tx=…`——原模板 "TRX/yr" 标注失实（CurrentYield/TargetYield 实为年化百分数，pp 差=Target−Current）；tx 短 hash 附带供审计（扩形）。
  - 引擎异常/停驶：全部失败/降级径（领奖败/改投败/带宽跳/采集败/账户查询失败·恢复/装配降级三项）kind=`trxvote_engine`——装配降级=boot 边沿每因至多一条；账户查询沿用既有 episode 边沿；运行期重复靠 Alerter 同指纹 5→60min 退避梯。`vote_enabled=false` 停驶待令=设计态不推。
  - 能量租金：`TxsCollector.SetOnRentalIncome` 回调（增量段新行才发——首拉/全史回填静默防轰炸），接线侧 ≥10 TRX/笔门槛（微付款不推），kind=`trx_rent` `[能量租金] +28.34 TRX`。
- **kindSpec 四档**（humanize.go）：`trxvote_claim`/`trxvote_rebalance`/`trxvote_engine`/`trx_rent`——actable=false（业主三域外降级 [经营事件] 级，推送不缺位）。
- **不推面**（施工件③）：CMC 隔夜息/普投心跳/微付款/CMC WS 心跳零触——租金回调仅 rental_income kind 且 ≥10 TRX。
- **数据流正本**：docs/数据流.md 运维面块补 TRX 自营盘 feishu 推送一行（新消费径同 commit 更新律）。

### 测试证据

- go build ✓ / go vet（改动四包）✓ / gofmt -l 空 ✓
- 定向：`go test ./internal/portfolio/trxvote/... ./internal/data/trx/... ./internal/platform/notify/... ./cmd/core/...` 全 ok——新断言：`年化差+0.12pp`/`107.3k votes`/`≈$0.33` 折算段/无报价省略段/`trxvote_claim`·`trxvote_engine` kind 迁移/租金增量门三态+nil 钩（t22_rental_income_hook_test.go）
- 全门禁 scripts/check_all.sh 绿戳（commit 后对最终 HEAD 重跑落戳再推）

### 偏离说明

- 施工序票面「worktree t/0217」系撞号让位残文——按业主钦定单号走 t/0220（已在票内加注）。
- formatRebalanceAlert 取**重写**（票面授权「复用扩形或重写由你裁」）：原模板收益单位标注失实且缺票数面。
- 改投推送附 tx 短 hash（票面示例无）——审计对账需 tx 锚，扩形不偏语义。
- 验收锚①②（18:00Z 窗领奖推送/异常注入推送）为生产实证项：待决策席验收合并+部署后于 18:00Z 窗核销；施工面已备齐触发径（成功/失败/降级三路均接推送）。
