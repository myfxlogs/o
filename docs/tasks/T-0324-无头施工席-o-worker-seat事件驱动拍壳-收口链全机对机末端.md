# T-0324 无头施工席——o-worker-seat 事件驱动拍壳（收口链全机对机末端）

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` devin @ 2026-11-04（业主令——「派单应派无头施工席非 O-Dev pane」；收口链全机对机末端缺实证） |
| 领取 | （空=未领） |
| 优先级 | 高——决策席裁毕无机径派单=常径断（今日实证：派单只能靠人 pane 接力） |
| 涉及面 | `deploy/o-worker-seat.service`（新）、`scripts/meta/dev_tick.sh`（新）、`cmd/eventbus/main.go`（+dev 源）、`cmd/metabus/main.go`（+wake 映射）、`skills/施工席/SKILL.md`（新职卡）、`docs/数据流.md`（链路图同 commit 更新） |

## 背景（实证在案，勿再诊）

收口链现状：收口→裁决全机对机，唯**派单出口仍须踢人 pane**——O-Dev
是交互 devin 会话（tmux pane），pane 不开=派单无人接，跟踢 O-DM
同型脆弱。业主令定案：常径派单目标=无头施工席。

决策席第六刀已立全套可照铸的形：`o-decision-seat.service`(oneshot)
→`dm_tick.sh`（闸+devin -p 批拍+ops_headless 心跳）；metabus
`wake[kind]→unit`；eventbus `srcs` ops_event 水印帧。本票=同构第四席。

## 目标

```
席裁毕 → report.sh 派单 T-NNNN → ops_event 派单行 → eventbus dev 帧
  → metabus → systemctl --user start o-worker-seat（--no-block 形）
  → 无头施工席：闸查未领派单→领单→施工→收口/移交自报 → dm 帧 → 决策席
```

全链零 pane；O-Dev pane 退为人工观察窗/手动施工位，退出派单链。

## 修法（钦定形，勿改计）

- `deploy/o-worker-seat.service`：照 `o-decision-seat.service` 逐字对形
  ——Type=oneshot、ExecStartPre mkdir /var/tmp/o-worker、
  ExecStart=dev_tick.sh、NoNewPrivileges=true。
- `scripts/meta/dev_tick.sh`：照 dm_tick.sh 对形——
  - env：`export O_SEAT=施工席`（report.sh 自识别正标）；FIN_DSN 同
    `DM_ENV_FILE` 径读 env。
  - 闸（T-0315② 最新刻集口径同式）：每票 `max(ts)` 事件集含 `派单`
    且其后同票无 `领单`/`收口` → 活在；零活→`hb alive`+exit（不烧空拍）。
  - 拍：`devin -p --model swe --respect-workspace-trust false
    --permission-mode dangerous -- "读 /opt/o/skills/施工席/SKILL.md
    施工协议节 拍未领派单"`（cwd=/var/tmp/o-worker scratch）。
  - 心跳：每拍 ops_headless 行（ticket=META-Dev、seat=施工席、
    state=alive|stalled）。
- `cmd/eventbus/main.go`：srcs 加 `dev` 源——
  `WHERE event='派单'` 的 `max(id)` 水印（照 "d" 行对形）。
- `cmd/metabus/main.go`：wake 加 `"dev":"o-worker-seat.service"`——
  唤醒调用须异步形（T-0319 绞死案：activating unit 上同步 start
  绞死帧循环——若 T-0319 未合，本行落时顺手同避，勿再踩）。
- `skills/施工席/SKILL.md`：新职卡——
  - 领单协议：`report.sh 领单 T-NNNN` 同拍→`git worktree add
    /opt/o-wt/tNNNN -b t/NNNN-slug`→`bash scripts/provision_worktree.sh`；
  - 施工协议：票面自含读单（勿再诊）→实现→自测→commit→`report.sh
    收口`+`移交`（report.sh 必调——漏调=机制失效案 T-0317 在案；
    「收口」字样 commit 另有 post-commit 钩兜底）；
  - 禁线：超票面范围停、红线三件停（资金扩投/新平台/席位立换）、
    凭据域停（app_secret 等）、三轮自试不过→`report.sh 挂起`；
  - 写面窄白名单：分支/worktree/票面回填/report.sh 三通/ops_event。
- `docs/数据流.md`：链路图加 `dev 帧→o-worker-seat` 段（同 commit）。

## 验收标准

- [ ] `report.sh 派单 T-9999 test` → dev 帧→o-worker-seat 实激活
  （journal 起停迹+ops_headless META-Dev 心跳行双侧）。
- [ ] 闸零活时 dev_tick 空拍 `hb alive`+exit 绿（不烧空拍）。
- [ ] 无头施工席对测试派单跑全生命周期：`领单→收口→移交` 三行自报
  齐（relay/ops_event/票面三面对拍）。
- [ ] 自报 seat=施工席正标（非「无头施工席」兜底名）。
- [ ] O-Dev pane 不再收常径派单。

## 范围外

- 不改决策席裁决协议本体、不改 O-Dev pane 用途（人工观察/手动施工
  仍可用——只是退出派单链）。
- 不动 metabus 看门狗/eventbus 轮询本体、不动 report.sh 踢径
  （T-0323 域）。
- T-0319（systemctl start --no-block）同碰 `cmd/metabus/main.go`——
  后到者 rebase 前件，勿并行改同 hunk。
