# T-0323 裁毕踢交互席——report.sh REPORT_KICK 第四通道

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` devin @ 2026-11-04（业主令——T-0318 裁毕踢飞书未踢交互席实证缺） |
| 领取 | （空=未领） |
| 优先级 | 高——裁决完成不激活交互席=编排停拍等傻轮询（今日实证 25min 空轮） |
| 涉及面 | `~/bin/report.sh`（共器件——加可选径不动默认行为）、`scripts/meta/dm_tick.sh`（export 两行） |
| 修订 | `[角色:决策]` 2026-11-04——业主批「一席两形态」案（collab.md §1）：REPORT_KICK 语义定死=通报/升件径，**新增 REPORT_DISPATCH=O-Dev 常径派单径**——票由「踢交互席」扩为「踢径双目标」 |

## 背景（实证在案，勿再诊）

无头决策席裁毕走 `report.sh` 三通：relay.log+飞书+ops_event——**独缺踢
交互席**。T-0318 裁词 10:02 已落 relay.log，交互决策席（O-DM pane）
傻轮 service 态 25min 未见——业主指出「踢了 meta 的飞书没踢我」。

径已实证：O-DM pane=devin 交互会话（pane_pid 1078214=本会话族谱亲证），
`tmux send-keys -t O-DM` 测试踢已进 pane 生效（输入即激活）。

## 修法（钦定形，勿改计）

- `report.sh`：写完 relay.log 行后加踢径（两目标共用同一 send-keys
  机制，按事件选目标）——
  - 通报/升件径：`[ -n "${REPORT_KICK:-}" ] && tmux has-session -t
    "$REPORT_KICK" 2>/dev/null && tmux send-keys -t "$REPORT_KICK" -l
    "<自报行全文>" && sleep 1 && tmux send-keys -t "$REPORT_KICK" C-m`
  - 常径派单径：新事件 `派单`——`report.sh 派单 T-NNNN "<转发文>"`，
    自报行照旧落三通，踢径目标改 `${REPORT_DISPATCH:-}`（同一段
    send-keys 代码，目标变量换名）。
  - **Enter 键名在 devin TUI 不注册——必须用 `C-m`**（11-04 双侧
    实证：Enter 滞留输入行、C-m 真提交）。失败静默同现有纪律。
- `dm_tick.sh`：头部两行——`export REPORT_KICK=O-DM`（自报通报/升件
  自动踢交互席）+ `export REPORT_DISPATCH=O-Dev`（席 `report.sh 派单`
  直踢施工 pane）。
- 其余席不设默认——谁要踢谁在自己 env 里开（可选径不污染meta席）。

## 验收标准

- [ ] `REPORT_KICK=O-DM report.sh 收口 T-9999 test` 实测进 O-DM pane
  （pane 可见行=实证）。
- [ ] `REPORT_DISPATCH=O-Dev report.sh 派单 T-9999 test` 实测进 O-Dev
  pane 且 `ops_event` 落 `派单` 行（双证）。
- [ ] 不设 REPORT_KICK/REPORT_DISPATCH 时行为与今全同（零回归）。
- [ ] dm_tick 下拍裁决行实踢实证（journal+pane 双侧）。

## 范围外

- 不改 report.sh 三通既有行为、不动静默纪律。
- 不动 metabus/eventbus/钩。
- 席「何时派单」的判据归决策席 SKILL/职卡（队列序+返修子票白名单）——
  本票只管踢径机制与双目标传输。
