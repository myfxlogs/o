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

## 结果回填

- ①venue 分子改：open 持仓 notional+在途 intent（fin_execution state='open'
  已发令牌未成交）——armed 未开仓策略不计（额度=真实敞口上限非管道上限；
  0 仓时 vExp≈0 不再拒开）。exposure_cap 同径并轨。
- ②armed 超龄退役：retireStaleArmed 加龄闸——fin_proposal armed 超
  lifecycle.armed_ttl_h（缺省 24h）未开火→fin_retired+notes 留痕；
  fin_strategy armed 同径 retired（额度释放+T-0069 淤积同族治）。
- ③拒放冷窗：denyAt map 记策略最近拒放戳——lifecycle.reject_cooldown_s
  （缺省 300s）冷窗内跳过同策略（8705/15min 空转止血）。
- 机检：build✓ gofmt✓ test✓（lifecycle+risk 绿）；棘轮零触。

## 返修①（A/B 增量）

- A：策略退役谓词收紧——`live/probe` 态+无 open 持仓+最近活动戳
  （fin_position 末次开仓 vs entered_at 兜底）超 armed_ttl_h 才退役；
  有仓/近期开过的不再误杀（原 state='armed' 裸谓词替换）。
- B：denyAt 拒放出口全位接线——lifetimeOK/breakevenOK/op.Open 三拒径
  同写戳；冷窗查前置保险丝检查前（窗内 skip 早退——拒因级均匀覆盖）；
  lifecycle.reject_cooldown_s 缺省 300s 热启。
- 修误：fin_proposal 无 notes 列——退役 UPDATE 去 notes 写（桩测红实证）
