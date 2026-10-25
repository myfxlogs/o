# T-0128 guard 判据链补 tick 新鲜度闸（高危——误杀实证在案）

| 字段 | 值 |
|---|---|
| 状态 | 待派 |
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
