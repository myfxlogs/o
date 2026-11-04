# T-0324 无头施工席——拉起式施工 run（收口链全机对机末端）

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` devin @ 2026-11-04（业主令——「派单应派无头施工席；席派单时拉起无头 devin（模型 swe-2-high|medium）做施工，完成回交无头决策席审计验收」） |
| 领取 | （空=未领） |
| 优先级 | 高——决策席裁毕无机径派单=常径断（今日实证：派单只能靠人 pane 接力） |
| 涉及面 | `scripts/meta/dev_run.sh`（新）、`skills/施工席/SKILL.md`（新职卡）、`skills/决策席/SKILL.md`（形态域界补拉起式一行）、`scripts/meta/dm_tick.sh`（尾段漏拉清扫闸）、`docs/数据流.md`（链路图同 commit 更新） |
| 修订 | 2026-11-04 二版——弃常驻 o-worker-seat.service+eventbus dev 源+metabus 映射方案，改业主钦定**拉起式**：席派单同拍 `systemd-run` 起瞬态施工 run |

## 背景（实证在案，勿再诊）

收口链现状：收口→裁决全机对机，唯**派单出口仍须踢人 pane**——O-Dev
是交互 devin 会话（tmux pane），pane 不开=派单无人接。业主令定案：
施工席无头/交互并存不打架（票域独占天然隔离），常径派单拉起无头
devin run，模型钦定 `swe-2-high` 或 `swe-2-medium`（High|Medium 二择，禁他档）；交互 O-Dev pane 退为人工观察窗/
手动施工位，退出派单链。

机制底件全实测在案：`systemd-run --user --unit=X --collect` 起瞬态
unit 逃逸父 cgroup（oneshot 退出不杀子——10:27 o-worker-probe 实证），
`devin -p --model swe-2-high` 旗值在册，收口/移交行产 dm 帧唤决策席
既有径。

## 目标

```
席裁毕 → report.sh 派单 T-NNNN（ops_event 派单行落账）
       → 同拍 systemd-run --user --unit=o-worker-<tid> --collect
         dev_run.sh <tid> "<转发文>"
       → 无头施工 run（swe-2-high|medium）：领单→施工→收口/移交自报
       → dm 帧 → o-decision-seat → 审计验收
```

全链零 pane 依赖；派单行落账=可审计锚，拉起=即行不候帧。

## 修法（钦定形，勿改计）

- `scripts/meta/dev_run.sh`：施工 run 壳——
  - `export O_SEAT=施工席`（report.sh 自识别正标；FIN_DSN 照
    dm_tick.sh 同径读 `REPORT_ENV_FILE`）；
  - `mkdir -p /var/tmp/o-worker && cd` 之（scratch 零仓件——worktree
    由职卡领单步自建）；
  - `devin -p --model <档> --respect-workspace-trust false
    --permission-mode dangerous -- "<提示词>"`——档钦定二择：
    `swe-2-high`（默认）或 `swe-2-medium`，禁他档；
  - 收尾 ops_headless 心跳行（ticket=`META-Dev-<tid>`、seat=施工席、
    state=alive|stalled——**按票分行**，并行施工 run 互不盖心跳）。
- `skills/施工席/SKILL.md`：新职卡（无头施工协议）——
  - 领单四步序：`report.sh 领单 T-NNNN`→`git pull`→`git worktree add
    /opt/o-wt/tNNNN -b t/NNNN-slug`→`bash scripts/provision_worktree.sh`；
  - 施工：票面自含读单勿再诊；diff ⊆ 涉及面；红线三件/凭据域/改票
    本体三禁；自测 build+vet+test+check_migrations 全绿才收口；
  - 收口三步同拍：票面回填翻待验→`git commit -o -- <路径>`（共享
    index 坑在案）→`report.sh 收口`+`移交`（必调——漏调断链案
    T-0317 在案，「收口」字样 commit 另有 post-commit 钩兜底）；
  - 停与挂：三轮自试不过/外因阻塞→`report.sh 挂起`；熔断级→`异常`。
- `skills/决策席/SKILL.md` 形态域界节补一行钦定拉起式：
  `report.sh 派单` 落行同拍——
  `systemd-run --user --unit="o-worker-<tid>" --collect
   /opt/o/scripts/meta/dev_run.sh <tid> "读 /opt/o/skills/施工席/
   SKILL.md 施工协议 领 <tid> 施工"`；禁踢 O-Dev pane（派单非通报）。
- `scripts/meta/dm_tick.sh` 尾段加漏拉清扫闸：ops_event 中 `派单`
  行后同票无 `领单`且无活 `o-worker-<tid>` unit → 以 dev_run.sh
  补拉（幂等——席 run 半路死漏拉的机制兜底，T-0317 漏调案同型
  教训：兜底靠机制不靠记性）。
- 同段再加**收口对拍闸**（缺席发现器定理——坑条目
  absence-detector-missing-report-rows）：扫在飞 t/* 分支新
  commit，票号可提且 ops_event 超 15min 无同票收口行→代录收口行
  （note 标 `relay@hook-miss`）+踢 O-DM 知会——漏调/钩失配的
  最后一层兜底，人肉搬运退为第三层。
- `docs/数据流.md`：链路图加「席派单→o-worker-<tid> 瞬态 run」段
  （同 commit）。

## 验收标准

- [ ] `report.sh 派单 T-9999 test`+拉起实证：`systemd-run` 起
  `o-worker-t9999` 瞬态 unit（journal 起停迹+META-Dev-t9999 心跳行）。
- [ ] 施工 run 对测试单跑全生命周期：`领单→收口→移交` 三行自报齐
  （relay/ops_event/票面三面对拍），自报 seat=施工席正标。
- [ ] 漏拉清扫闸实证：手工落 `派单` 行不拉起→dm_tick 下拍补拉。
- [ ] 并行两单施工互不打架（心跳分行+票域独占实证）。
- [ ] O-Dev pane 不再收常径派单。

## 范围外

- 不改决策席裁决协议本体；不改 O-Dev pane 用途（人工观察/手动施工
  仍在——只是退出派单链）。
- 不动 eventbus/metabus/report.sh 踢径（T-0323 域）；不动 T-0319
  （no-block 修）。
- 施工席职卡只管无头协议——O-Dev pane 的人工会话不受其约束。
