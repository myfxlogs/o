# T-0323 返修①——踢径归钦定单目标+O_SEAT 销误标（规漂非责：收口后 2 分钟票面修订②落）

驳回由（决策席无头拍 devin @ 2026-11-04）：收口件 @6494f45e（分支
t/0323-319-wake-relay）按修订①（10:22 双目标钦定）施工在案——非
施工过错；修订②（10:26 业主令——REPORT_DISPATCH 概念作废回归单
目标、派单走 ops_event 行→dev 帧→o-worker-seat）收口后落 master。
按现行票面核，两违一缺：

- **违①**：`dm_tick.sh` `export REPORT_DISPATCH=O-Dev`——激活业主令
  明禁的「派单踢 O-Dev pane」径（O-Dev 是交互件须人开，已退派单链）。
- **违②**：`~/bin/report.sh` 派单事件分流 `REPORT_DISPATCH` 第二
  目标——钦定形「不设 REPORT_DISPATCH 类第二目标；一切事件同目标，
  含派单自报也踢 O-DM 通报」。
- **缺③**：钦定两行缺一——`export O_SEAT=决策席` 未落，误标实证
  不销（O_SEAT 空时 report.sh 兜底「无头施工席」+心跳错打 hb Dev，
  relay 10:02-10:04 五行在案）；锚④「自报 seat=决策席」无证。

在案面：门禁亲跑全绿（go build/vet ./cmd/metabus、
check_import_direction、check_migrations）；范围红线合（metabus/
dm_tick/report.sh/两票文档俱在涉及面）；踢径机制实证在案
（ops_event 1493-1495）。T-0319 部（--no-block）合规在案**不动**，
随同件合。

## 返修须落

1. `scripts/meta/dm_tick.sh`：`export REPORT_DISPATCH=O-Dev` →
   `export O_SEAT=决策席`（`export REPORT_KICK=O-DM` 保留）——
   钦定两行形。
2. `~/bin/report.sh`：删 `派单→REPORT_DISPATCH` 分流分支，归钦定
   单目标——`KICK_TGT=$REPORT_KICK`，一切事件同目标（含派单自报
   行踢 O-DM 通报）。
3. 实证（沿用 T-9999 形）：
   - `REPORT_KICK=O-DM report.sh 收口 T-9999 返修验` 进 O-DM pane
     （ops_event 收口行+pane 行双证）；
   - 不设 REPORT_KICK 时行为与今全同（零回归）；
   - `O_SEAT=决策席 report.sh 派单 T-9999 返修验` → ops_event 派单行
     seat=决策席+hb DM（误标销实证；O-Dev pane 无踢行=明禁径未激活）。
4. 部署后锚照旧：dm_tick 下拍裁决行实踢（journal+pane 双侧）；
   T-0319 activating 灌帧——俱随下次 unit 部署链销。
5. 范围外照旧：不改 report.sh 三通既有行为/静默纪律、不动
   metabus/eventbus/钩；票面不动（修订②锚面已是终形）。
