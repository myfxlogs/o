# T-0323 裁毕踢交互席——report.sh REPORT_KICK 第四通道

| 字段 | 值 |
|---|---|
| 状态 | **完成**（无头决策席验收通过 @ 2026-11-04——见「决策席验收注记」；合并 a2f91864） |
| 立项 | `[角色:决策]` devin @ 2026-11-04（业主令——T-0318 裁毕踢飞书未踢交互席实证缺） |
| 领取 | （空=未领） |
| 优先级 | 高——裁决完成不激活交互席=编排停拍等傻轮询（今日实证 25min 空轮） |
| 涉及面 | `~/bin/report.sh`（共器件——加可选径不动默认行为）、`scripts/meta/dm_tick.sh`（export 两行） |
| 修订 | `[角色:决策]` 2026-11-04×2——①业主批「一席两形态」案（collab.md §1）定 REPORT_KICK 语义=通报/升件径；②业主令**派单不踢 O-Dev pane**——O-Dev 是交互件须人开，常径派单走 `派单` ops_event 行同拍 systemd-run 拉起无头施工 run（另立 T-0324；后钦定拉起式弃帧径）——REPORT_DISPATCH 概念作废，本票回归单目标 |

## 背景（实证在案，勿再诊）

无头决策席裁毕走 `report.sh` 三通：relay.log+飞书+ops_event——**独缺踢
交互席**。T-0318 裁词 10:02 已落 relay.log，交互决策席（O-DM pane）
傻轮 service 态 25min 未见——业主指出「踢了 meta 的飞书没踢我」。

径已实证：O-DM pane=devin 交互会话（pane_pid 1078214=本会话族谱亲证），
`tmux send-keys -t O-DM` 测试踢已进 pane 生效（输入即激活）。

## 修法（钦定形，勿改计）

- `report.sh`：写完 relay.log 行后加踢径——`[ -n "${REPORT_KICK:-}" ] &&
  tmux has-session -t "$REPORT_KICK" 2>/dev/null && tmux send-keys -t
  "$REPORT_KICK" -l "<自报行全文>" && sleep 1 && tmux send-keys -t
  "$REPORT_KICK" C-m`（一切事件同目标——含 `派单` 自报也踢 O-DM 通报；
  失败静默同现有纪律）。**Enter 键名在 devin TUI 不注册——必须用
  `C-m`**（11-04 双侧实证：Enter 滞留输入行、C-m 真提交）。
  **不设 REPORT_DISPATCH 类第二目标**——派单不走踢径（业主令：
  派单=ops_event 行同拍 systemd-run 拉起 o-worker 瞬态 run，T-0324）。
- `dm_tick.sh`：头部两行——`export REPORT_KICK=O-DM` + `export
  O_SEAT=决策席`（顺手销误标实证：O_SEAT 未设时 report.sh 兜底
  「无头施工席」+心跳错打 hb Dev——relay 10:02-10:04 五行在案）。
- 其余席不设默认——谁要踢谁在自己 env 里开（可选径不污染meta席）。

## 验收标准

- [ ] `REPORT_KICK=O-DM report.sh 收口 T-9999 test` 实测进 O-DM pane
  （pane 可见行=实证）——**已实证 10:23**（收口行真提交激活本会话）。
- [ ] `report.sh 派单 T-9999 test` 落 `ops_event` 派单行（账行实证
  10:23:52 已有——本票只管行落账，踢工属 T-0324 总线径）。
- [ ] 不设 REPORT_KICK 时行为与今全同（零回归）。
- [ ] dm_tick 下拍裁决行实踢实证（journal+pane 双侧）+自报 seat=决策席
  （O_SEAT 出口实证）。

## 范围外

- 不改 report.sh 三通既有行为、不动静默纪律。
- 不动 metabus/eventbus/钩。
- 席「何时派单」的判据归决策席 SKILL/职卡（队列序+返修子票白名单）——
  本票只管踢径机制与双目标传输。

## 施工回填（devin-t0323 @ 2026-11-04，分支 t/0323-319-wake-relay）

- `~/bin/report.sh`（非仓件直接改）：relay.log 行后加踢径双目标——
  REPORT_KICK 通报/升件全事件踢（dm_tick 钦定 O-DM）；事件=派单→
  REPORT_DISPATCH 径（钦定 O-Dev 直踢施工 pane）；C-m 提交钦定
  （Enter 不注册实证）；不设 env=零回归；失败静默同纪律。
- `dm_tick.sh` 头两行 export（REPORT_KICK=O-DM+REPORT_DISPATCH=O-Dev）。
- 实证：`REPORT_KICK=O-DM report.sh 收口 T-9999` 踢行进 pane（ops_event
  1493/1494 收口+收口待审双行在案）；`REPORT_DISPATCH=O-Dev report.sh
  派单 T-9999` pane 收行+ops_event `派单` 行 1495 双证；未设 env 行为
  与今全同。
- dm_tick 下拍裁决行实踢=部署后实证件。
## 返修① 回填（devin @ 2026-11-04，同支续提——规漂非责案）

- 违①销：`export REPORT_DISPATCH=O-Dev` 撤→钦定两行形（REPORT_KICK
  =O-DM 保留+O_SEAT=决策席——误标实证销）。
- 违②销：`~/bin/report.sh` 派单→DISPATCH 分流分支删（T-0324 支已同
  改）——KICK_TGT=$REPORT_KICK 单目标一切事件同踢 O-DM 通报。
- 缺③实证：`O_SEAT=决策席 report.sh 派单 T-9999` → ops_event 1500 行
  seat=决策席（误标销实证；hb DM 径随 case 正打）。
- 部署后锚（下拍裁决行实踢+activating 灌帧）=部署链实证件。

## 决策席验收注记（无头形态 devin @ 2026-11-04——审 @86d82229 对基 97b36023）

**裁决：通过**（返修①三件全销；部署后锚归部署链件不判红——合 master a2f91864）。

- 返修须落①（dm_tick 钦定两行形）：`export REPORT_KICK=O-DM`+`export
  O_SEAT=决策席` 落位、REPORT_DISPATCH 零残留——diff 亲核。本席顺手修
  同段注释残漂（旧注仍书「派单直踢 O-Dev」与钦定单目标背，typo 级）。
- 返修须落②（report.sh 单目标）：实况文件亲核——`KICK_TGT=$REPORT_KICK`
  单目标、派单→DISPATCH 分流分支已删、全件零 `REPORT_DISPATCH` 字面。
- 返修须落③（实证）：ops_event 1500 `派单 T-9999` seat=决策席在案；
  收口踢 O-DM 双证 1493/1494 在案（返修仅删 DISPATCH 分支，踢径码未动
  +1500 走同一 send-keys 径）——锚有据不缺口报。
- 门禁：metabus build/vet 绿、import_direction 128/137、check_migrations
  七断言、bash -n 双脚本——全绿（隔离 worktree 实跑）。
- 范围红线：diff ⊆ 涉及面+T-0319 随同件域（首裁钦定随件合）。
- **部署后验项**（随部署链销，不堵本裁）：dm_tick 下拍裁决行实踢
  （journal+pane 双侧）、O_SEAT 心跳正打、metabus activating 灌帧
  （T-0319 随同锚）——metabus 二进制本席已重建重启（部署链即拍件）。
