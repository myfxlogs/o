# T-0148 okxws 拒因门键 depth5_no_ctval 未注册（拒门完整性漏）

## 态：完成（判过并码在案，11-01 清账翻面·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**完成**——审计通过落票在案，漏网翻面
## 审计席：headless @ 2026-10-30
## 领取：zcode-o-7d3f @ 2026-10-26
## 域：桥面数据面 / rejects 审计门
## 源出：T-0142 部署窗首见，T-0144 验证窗仍在——o-core 日志每分 ~77 行
  `WARN rejects[okxws] 未注册门键 "depth5_no_ctval"`（05:50 实拍，占日志量 ~96%）

## 病灶
depth5 蒸馏径 ctval（contract value）缺字段拒因 `depth5_no_ctval` 未入
rejects 预注册表——每次 Inc 打 WARN。功能上数据仍如实缺席（fail-closed
正确），但审计门完整性破+日志洪泛掩真信号（xvenue_carry 评估行被淹）。

## 施工
- `depth5_no_ctval` 注册进 rejects 门键表（okxws 域）——消 WARN 留计数。
- 顺带普查：grep 全仓 `rej.Inc("..."` 键 vs 注册表对账，同族漏注册一并清。
- 桩锚：depth5 缺 ctval 行→计数+零 WARN。

## 验收锚
- o-core 日志该 WARN 绝迹；rejects 计数器 depth5_no_ctval 正常累加。
- 全仓 Inc 键与注册表机检对账零漏（考虑入 check_ 闸防再犯——可并入
  既有 rejects 完整性断言如有）。

## 结果回填

- ①depth5_no_ctval 注册进 okxws 门（collector.go NewCollector）——消
  WARN 留计数。
- ②顺带普查（跨行感知全仓静态对账，非测试面）：真漏共 8 键跨 5 包，
  一并清——okxws:depth5_no_ctval（本票主灶）、lifecycle:facts_missing/
  missing_param/oscillation_fuse（opener.go 判据缺/参数缺/振荡保险丝）、
  hypothesis:sig_missing_params/sig_write_err（signals.go）、
  sigdist:xvenue_map_missing（xvenue.go）、earn:transfer_rejected
  （mover.go）。
- ③动态拒因面核毕：risk check.go:148 `g.rej.Inc(reason)`（deny 径）
  静态不可判——deny 实参取值域十值逐一亲证全在注册表（equity_missing/
  exposure_cap/intent_incomplete/kill_switch/missing_param/notional_cap/
  venue_concentration/token_expired/token_missing/token_used），零漏。
- ④机检入闸：scripts/check_rejects_keys.sh（全域一致性检查，pre-commit
  6c 位）——各包 `.Inc("字面量")` ⊆ 同包 rejects.New 注册集，漏注册=红；
  动态 Inc 列 INFO 留人审。对抗验证：撤 sigdist 注册键即红、复原即绿。
- 桩锚（internal/data/okxws/rejects_test.go）双绿：TestOkxwsDoorHas
  Depth5NoCtval（生产门构造直绑——注册漏即红）+TestDepth5NoCtvalCount
  NoWarn（缺 ctval 行二帧计数累加=2+log 捕获零「未注册门键」WARN）。
- 机检：build/vet/gofmt/staticcheck(okxws+rejects)✓ 六触面包 go test
  -count=1 全绿（okxws 7.0s/lifecycle 86.5s/hypothesis 44.1s/sigdist
  55.7s/earn 96.4s）；闸脚本 OK 零漏。
- 偏离说明：普查发现本票病灶非孤例——同族 7 键随票面「一并清」条款
  清掉，非越单。
- 生产实证锚（待验席部署窗亲证）：o-core 日志 `WARN rejects[okxws] 未
  注册门键 "depth5_no_ctval"` 绝迹+rejects 快照面 `okxws/depth5_no_ctval`
  正常累加（status 端点 Snapshot 同源）。

## 审计注记（headless @ 2026-10-30，逐锚对账·全实证非引述）

- **锚①（该 WARN 绝迹+depth5_no_ctval 正常累加）：PASS**——机制面：
  rejects.Door.Inc 未注册键→`log.Printf("WARN …未注册门键")`、注册键
  →原子加（internal/platform/rejects/rejects.go:25-31），键注册后
  WARN 径构造性不可达。桩锚真跑：TestOkxwsDoorHasDepth5NoCtval（生产
  门 NewCollector 构造直绑——注册漏即红）+TestDepth5NoCtvalCountNo
  Warn（books5 缺 ctval 帧×2→Snapshot["okxws/depth5_no_ctval"]=2+
  log 捕获零 WARN）双绿（2 passed 实拍）。生产面：o-core 容器末次运行
  2026-10-29 14:35:10–14:36:57Z（107s 窗、config gen=286、全量日志
  53 行）零「未注册门键」——对照票面历史洪泛 ~77 行/分，同流同率下
  该窗应见 ~137 条，实测零条=与修复上线态一致；live 计数为 in-memory
  Door 随停驶不可复拍（票面已自明部署窗亲证口径，机制面证毕）。
- **锚②（全仓 Inc 键⊆注册表机检对账零漏+入 check_ 闸）：PASS**——
  scripts/check_rejects_keys.sh 实存（git mode 100755），pre-commit
  6c 位入链（scripts/hooks/pre-commit）。本席对抗双证复跑：修复前基点
  abe5dd60^ 跑闸=红，精确列 8 漏键跨 5 包（okxws/depth5_no_ctval、
  earn/transfer_rejected、lifecycle/facts_missing·missing_param·
  oscillation_fuse、hypothesis/sig_missing_params·sig_write_err、
  sigdist/xvenue_map_missing）——与回填普查口径逐键吻合；origin/master
  跑=绿「零漏」，动态 Inc（internal/risk/check.go:148 `g.rej.Inc(
  reason)`）正确归 INFO 留人审；deny 实参十值域逐一核全在注册表。
- **宣称件三证齐**：注册点五处 diff 实见+现文件实存（okxws
  collector.go/earn worker.go/lifecycle run.go/hypothesis run.go/
  sigdist run.go）；Inc 调用点 grep 实证（okxws collector.go:204·
  handle.go:59、earn mover.go:110、lifecycle opener.go:104·167-229、
  hypothesis signals.go:22-102、sigdist xvenue.go:53）；桩锚两测试
  实跑 2 passed；闸脚本双向行为与宣称符（红→绿、动态列 INFO）。
- **边界注记**：o-core 现停驶（Exited ~24h），生产日志锚取证限于末次
  107s 窗+机制面——部署窗深度亲证属验席口径（票面目明）；机制面对已
  注册键 WARN 径构造性不可达，判决不因此降级。
