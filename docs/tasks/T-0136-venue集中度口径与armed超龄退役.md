# T-0136 venue 集中度口径修+armed 超龄退役（高危——开仓死锁实证）

## 实证（SG 2026-10-26）

- 0 持仓 0 开仓；`fin_strategy` 非退役 269 行合计 notional=**$13,500**（okx 单 venue）
- `venue_concentration` 闸口径=非退役策略 notional 总和 vs equity×maxPct（~$3.5k）——armed 未开火策略占位预算已超额，任何开仓必拒（生产日志 venue_concentration 拒放在案）
- 旁证：1017 条 armed、15min 8705 次拒放搅动——armed 淤积病灶与 T-0069 同族复发

## 修法

1. `risk/check.go` venue 集中度口径：分子改「open 持仓 notional + 在途 intent（已发令牌未成交）」——armed 策略**开火时竞额度**而非注册时占死（额度本义=真实敞口上限非管道上限）
2. armed 超龄退役：armed 提案/策略创建超 N 小时（建议 24h，键 `lifecycle.armed_ttl_h`）仍未过闸开火→自动 `retired`+留痕（释放预算+治 T-0069 淤积同族）；退役行写清 reason
3. 拒放搅动冷却：同策略连拒同因入短冷窗（建议 5min，键 `lifecycle.reject_cooldown_s`）——8705 次/15min 空转止血

## 验收锚

- 0 持仓下 venue_concentration 不再拒（vExp=0）
- armed 超龄行自动翻 retired、留痕 reason 可查
- 拒放日志频次降至冷窗量级
- 单测：armed 在册未开仓不计 venue 分子；超龄退役触发

## 返修①（决策席 2026-10-26——@068ede0 半交付打回）

- **缺 A**：策略侧退役 `WHERE state='armed'` 死代码——fin_strategy 无 armed 态（实证状态集 paper/probe/live/degraded/retired）。占位实体=`live` 态未开火策略（110 行）。修：退役谓词改「state IN ('live','probe') 且无 open 持仓 且 最近活动戳（entered_at/最近仓位 opened_at 取新者）超 armed_ttl_h」——有仓/近期开过的 live 策略不得误杀。
- **缺 B**：`denyAt` 声明未接线——同因同策略连拒入冷窗的判读点在拒放出口处查/写 denyAt，窗内直接 skip（键 `lifecycle.reject_cooldown_s` 缺省 300）。
- 保留已合格的 ①风控口径修+②提案侧 TTL——增量补 A/B 即可。
