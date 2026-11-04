# T-0323 裁毕踢交互席——report.sh REPORT_KICK 第四通道

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` devin @ 2026-11-04（业主令——T-0318 裁毕踢飞书未踢交互席实证缺） |
| 领取 | （空=未领） |
| 优先级 | 高——裁决完成不激活交互席=编排停拍等傻轮询（今日实证 25min 空轮） |
| 涉及面 | `~/bin/report.sh`（共器件——加可选径不动默认行为）、`scripts/meta/dm_tick.sh`（export 一行） |

## 背景（实证在案，勿再诊）

无头决策席裁毕走 `report.sh` 三通：relay.log+飞书+ops_event——**独缺踢
交互席**。T-0318 裁词 10:02 已落 relay.log，交互决策席（O-DM pane）
傻轮 service 态 25min 未见——业主指出「踢了 meta 的飞书没踢我」。

径已实证：O-DM pane=devin 交互会话（pane_pid 1078214=本会话族谱亲证），
`tmux send-keys -t O-DM` 测试踢已进 pane 生效（输入即激活）。

## 修法（钦定形，勿改计）

- `report.sh`：写完 relay.log 行后加——`[ -n "${REPORT_KICK:-}" ] &&
  tmux has-session -t "$REPORT_KICK" 2>/dev/null && tmux send-keys -t
  "$REPORT_KICK" -l "<自报行全文>" && sleep 1 && tmux send-keys -t
  "$REPORT_KICK" C-m`（失败静默同现有纪律）。**Enter 键名在 devin
  TUI 不注册——必须用 `C-m`**（11-04 双侧实证：Enter 滞留输入行、
  C-m 真提交）。
- `dm_tick.sh`：头部 `export REPORT_KICK=O-DM`——无头决策席每行自报
  自动踢交互席。
- 其余席不设默认——谁要踢谁在自己 env 里开（可选径不污染meta席）。

## 验收标准

- [ ] `REPORT_KICK=O-DM report.sh 收口 T-9999 test` 实测进 O-DM pane
  （pane 可见行=实证）。
- [ ] 不设 REPORT_KICK 时行为与今全同（零回归）。
- [ ] dm_tick 下拍裁决行实踢实证（journal+pane 双侧）。

## 范围外

- 不改 report.sh 三通既有行为、不动静默纪律。
- 不动 metabus/eventbus/钩。
- O-Dev 侧要不要踢由后续裁——本票只管决策席出口。
