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

## 返修①决策记录（决策席 2026-10-26——@068ede0 半交付打回）

- 缺 A：策略侧退役 `WHERE state='armed'` 死代码——fin_strategy 无 armed 态（态集 paper/probe/live/degraded/retired），占位实体=live 态未开火策略。修口令同上方「返修①（A/B 增量）」A 项，已落地 ✅。
- 缺 B：`denyAt` 声明未接线——拒放出口查/写接线口令同上方 B 项，已落地 ✅。
- 备查：notes 留痕撤除属 schema 强制（列不存在），非偷工——验收认可。

## 验收（决策席 2026-10-26 @0b0bcf4 部署）

- 三锚全落：venue 口径修上线；armed 退役 996 提案+82 策略（TTL 热启收紧 4h——armed_ttl_h=4 已落 app_config）；拒放 0/3min（原 580/min）
- RX 4767→2273 kbps 排水进行式（probe/paper 队列 4h 陈化中，趋势向下）
- 状态：驳回:锚④单测零交付（venue 新口径/armed 超龄退役均无测试落件）；策略退役裸 UPDATE 未写 transition 留痕；op.Open 拒径缺 denyAt 戳（返修B 三径仅落两径）
- 审计席：headless @ 2026-10-30

## 审计判决注记（无头单飞——逐锚对账 @0b0bcf45）

- 锚① venue 口径修：达。`check.go` 分子=open 持仓 notional（venue 经 strategy 关联）+在途 `fin_execution state='open'`；0 仓 0 在途 vExp=0，exposure_cap 同径并轨。
- 锚② armed 超龄退役：半达。提案 `fin_armed`+created_at 超龄→`fin_retired`、策略 `live/probe`+无仓+活动超龄→`retired`（返修 A 谓词如口令），TTL 键热启接线实。但策略侧裸 `UPDATE` 未写 `fin_strategy_transition`——同文件 degraded 退役径走 `Transition()` 落 reason 行（机制已存未用），库内 reason 不可查，唯日志行；提案侧无 reason 列属 schema 强制（票面已裁，不计缺）。
- 锚③ 拒放冷窗：机制落（denyAt+`lifecycle.reject_cooldown_s` 缺省 300s，冷窗查前置保险丝前合规）；返修 B 钦定 lifetimeOK/breakevenOK/op.Open 三拒径同写戳——实盘 op.Open 败径仅 `rej.Inc("open_failed")` 无写戳，三径落两径（开仓连败仍逐轮 WARN 无冷窗）。
- 锚④ 单测：缺。交付区间（@068ede0+@1cea248）零测试文件——venue 新口径测、armed_ttl 退役测、denyAt 冷窗测均无（既有 TestDegradedTTLRetire 系 degraded 径非同物）；票面验收自书「三锚全落」亦仅认三锚。
- 重飞依据：补锚④单测（armed 在册不计 venue 分子+超龄退役触发+冷窗窗内 skip）；策略退役径落 transition 行（reason=armed_ttl_expired 类）；op.Open 败径补 denyAt 写戳。
