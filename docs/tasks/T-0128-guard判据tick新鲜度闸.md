# T-0128 guard 判据链补 tick 新鲜度闸（高危——误杀实证在案）

| 字段 | 值 |
|---|---|
| 状态 | ✅ 验收合入部署 @f1bd7b8（tick/funding 龄闸+alerter 挂接+0055 迁移；返修①②全补）|
| 优先 | **高**——真仓误杀实证（#23），所有现货腿对冲仓暴露 |
| 出处 | 仓 23 delta_drift 误杀取证；坑条目 `guard-drift-stale-tick-false-kill` |

## 病灶

`internal/guard/guard.go` `notionalNow` 只断言 tick 行存在，不断言新鲜度。
md_ticks 状态表模型下行恒在、内容可陈——一腿陈一腿新即产生价时差型假
失衡。实证：现货 tick 死於 15:58、SWAP 新鲜，仓 23 17:00:02 被误杀
（真腿量仍平衡，基线 1.53bps）。

同病灶波及面：funding_flip 判据的 latestFunding/nextFunding 读法、
以及所有「行在即有效」的事实读面都应一并过一遍新鲜度口径。

## 修法方向（施工裁）

- `notionalNow`/事实读面统一加 age 断言（建议 ≤120s，可调键落
  `exec.tick_max_age_s` 类配置），陈行等同缺失→facts_missing 跳过
  （fail-closed，含「陈=缺」语义）。
- 可选增强：fin_guard_close 落 trigger 时的实测失衡值（bps）——取证面
  现在只有 trigger_kind，复盘要靠侧写。

## 验收锚

- 桩测：一腿 tick 陈（>阈值）一腿新→仓不杀+facts_missing 计数
- 两腿皆新但真失衡>band→照杀（不误伤）
- 部署后实证：现货 tick 断流窗口内无 delta_drift 触发记录

## 回执①（决策席 2026-10-26）

- **本体合格**：notionalNow tick 龄闸（120s 缺省可调）+fundingRate 龄闸
  （540min 对 8h 结算窗对径）+TestStaleTickNoKill 复现 #23 径——判据层
  「陈=缺」fail-closed 落地方向正确。
- **返修两缺**：
  1. 迁移撞号——0054 被 T-0131 proposal_dedupe_gate 先占，改 0055
     （文件名+登记行+头注三处同改，T-0109 机检拦实证）。
  2. staleWatch 面死告警仍 `log.Printf("ALERT")`——T-0117 同坑重犯：
     ALERT 不经 notify.Alerter+humanize kind 注册=业主收不到；collector
     需 AttachAlerter 挂接装配。

## 结果回填

- notionalNow 回读 ts+age 断言（exec.tick_max_age_s 缺省 120s——陈行=
  facts_missing 让位不杀；一腿陈一腿新不再产价时差型假失衡）。
- fundingRate 同径过闸（guard.funding_max_age_min 缺省 540min——结算窗
  8h 尺度）；0053 两键入 app_config 机读。
- 桩测：TestStaleTickNoKill（现货 4h 陈+SWAP 新→facts_missing 不杀）+
  TestFreshTickRealDriftKills（两腿皆新真失衡|imb|>500bps→照杀）PG 实跑绿。
- 坑条目 guard-drift-stale-tick-false-kill 直写 canon（仓 23 误杀实证）。

## 返修补丁（回执——迁移撞号+告警通道两件）

- ① 0054→0055 三处同步（文件改名+migrate_table 登记+头注）——master
  0054_proposal_dedupe_gate 先占；T-0127 小件索引用 0056。
- ② staleWatch 告警走 notify.Alerter humanize kind=okxws_face_dead
  （kindSpec 新增事件级行——「行情面断流」白话标题；nil=log 退路）；
  collector.AttachAlerter 装配件挂 cmd/main。
