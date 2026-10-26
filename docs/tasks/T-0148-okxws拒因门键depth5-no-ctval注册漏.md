# T-0148 okxws 拒因门键 depth5_no_ctval 未注册（拒门完整性漏）

## 态：待施工
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
