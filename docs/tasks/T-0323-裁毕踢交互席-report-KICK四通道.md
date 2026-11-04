# T-0323 裁毕踢交互席——report.sh REPORT_KICK 第四通道

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` devin @ 2026-11-04（业主令——T-0318 裁毕踢飞书未踢交互席实证缺） |
| 领取 | （空=未领） |
| 优先级 | 高——裁决完成不激活交互席=编排停拍等傻轮询（今日实证 25min 空轮） |
| 涉及面 | `~/bin/report.sh`（共器件——加可选径不动默认行为）、`scripts/meta/dm_tick.sh`（export 两行） |
| 修订 | `[角色:决策]` 2026-11-04×2——①业主批「一席两形态」案（collab.md §1）定 REPORT_KICK 语义=通报/升件径；②业主令**派单不踢 O-Dev pane**——O-Dev 是交互件须人开，常径派单走 `派单` ops_event 行→总线帧→无头施工席（另立 T-0324）——REPORT_DISPATCH 概念作废，本票回归单目标 |

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
