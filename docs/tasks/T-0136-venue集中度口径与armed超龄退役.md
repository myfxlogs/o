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
