# T-0285 施工 spawn 认领盲闸根修——带外径直通+交互席已领不可见双缺

| 字段 | 值 |
|---|---|
| 状态 | 作废（并入ADM总设计——绿地重建覆盖其意，管理席裁2026-11-01） |
| 立项 | `[角色:决策]` @ 2026-11-01（管理席令 `20261101-075433-MANAGE-3482`=obl38——门2-DM 落箱即点火） |
| 领取 | （session 标识 @ 时间；空=未领） |
| 优先级 | **高**——同票副本每发=~1.3M tokens 白烧+钱径双写风险靠 agent 自觉兜底（T-0283 实证在案） |
| 涉及面 | `/opt/meta/devin-headless-run.sh`（锁前预检段）、`/opt/meta/mbus-dispatch.sh`（route_directive_core/spawn_run 闸链同据预检）、`/opt/meta/devin-headless-role.md`（让位协议扩+带外禁径）、`docs/collab.md` §3（领单即推支钉）、meta/o 两仓票面；o 仓仅本票+KB 条目+回填 |
| 执行径 | 无头/交互皆可 |

## 背景

T-0283 双发实证（KB 条目 `headless-outband-spawn-interactive-claim-blind` 在案，**诊断锚齐勿重复勘查**）：

- 07:36:53 dispatcher 闸已拒（票面「已批」不在派发集）→ **07:48:04 带外手起 `systemd-run` 直 spawn `devin-headless-run.sh T-0283`**（scope 名/argv 形态/.spawned 无锚/dispatch 零行四证），绕全闸；
- 副本落入交互席「已领未推」窗（领 07:37→推支 07:51）：远端支判据假阴性初判「唯一在飞」，赖 worktree+真席途中推支才二次核实让位 rc=0 零写——烧 1.28M tokens；
- T-0256 flock=无头↔无头象限；交互席不入锁不持锁；带外 spawn 不持锚不经闸——两缺皆在机制外。

## 目标

任一 spawn 径（dispatcher 路由/返修/人工带外）起施工席前，「票已被领」四据——**远端支 ∨ `/opt/o-wt/t<NNNN>*` 工作树 ∨ 交互席进程相 ∨ Dev 箱 directive 已 ack**——任一命中即让位/不派，零 devin 调用；交互席「领单即推 claim 支」收窄未推窗；带外手起 wrapper 钉违例+归因痕。

## 钦定界

### ① 「已被领」判据组（两径同源一函数/一段同文，防判据漂移）

1. **远端支**：`git ls-remote --heads origin 't/<NNNN>-*'` 命中（网不可达降级=本地已抓 `refs/remotes/origin/t/<NNNN>-*`）；
2. **工作树**：`/opt/o-wt/t<NNNN>*` 存在且其 HEAD 支属 `t/<NNNN>-*`；
3. **进程相**：`devin`（acp|-p 两型）argv/cwd 锚同票（施工定锚宽——worktree 路径/t 支串/prompt 票号皆可为锚，pgrep 宽匹后须 `/proc/<pid>/cmdline` 实锤防 bash -c 误中，同 T-0256 兼容闸律）；
4. **Dev 箱消费相**：该票 directive 件已在 `mail/Dev/acked/`=交互席已接（辅证据——仅与 ①-③ 合判不单独立罪，覆盖「交互席未建树先动工」残余缝）。

### ② wrapper 锁前预检段（`devin-headless-run.sh`）

在 T-0256 兼容闸（pgrep 前代）同位加①组判据——命中即走让位四要件（共享 log 留行+`report.sh 异常`+MANAGE 报+rc=0），**零 devin 调用零 prompt 触碰**。spawn 行同段补 **caller 父链记录**（`ps -o ppid= -p $$` 逐层至 init，落首行）——带外可归因。

### ③ dispatcher 闸链（`mbus-dispatch.sh`）

`route_directive_core`/`spawn_rework` 派单前过①组——命中=`ob_gate_once` 不派不消费（行留 armed 监理可见），log 注「已被领证据=<命中据>」。

### ④ 纪律成文

- `collab.md` §3 补钉：「领单即推 claim 支——建支翻单 push **先于动工**，禁先施工后推」（现文「立即」无时序锚，T-0283 实证窗 14min）；
- `devin-headless-role.md` 补：「`devin-headless-run.sh` 只许 dispatcher/监理/票面明载径调用——席位手起=违例；救火 fallback 须先过①组预检并在让位/spawn 行注 spawn 源标」。

## 验收标准

- [ ] ①组四据桩测各一：造远端支/造 worktree/造交互进程锚/造 Dev 已 ack 件——wrapper 与 dispatcher 两径各让位/不派，log 行在案；
- [ ] 真票实证：对已领票（远端支在）手起 wrapper→秒让位零 devin 调用；
- [ ] 窗内实证：worktree 在+远端无支时手起 wrapper→工作树/进程相命中让位；
- [ ] `collab.md` §3 与 `role.md` 条款落字（领单即推支/带外禁径/caller 父链）；
- [ ] caller 父链在 spawn 首行在案（手起桩测行含父链）；
- [ ] 票面回填+`tools/kbgen gen` 索引重建。

## 范围外

- **不重派 T-0283**（真席在飞部署/收口待审中）；
- **不碰 DBP2/生产/DB**；
- T-0256 已覆盖面（flock 无头↔无头/锁前生代兼容闸/8>&-）不重立；
- audit wrapper 闸面不扩（沿用 T-0256 钦定界：审计席无 fin_* 写面）；
- 带外 spawn 席位归因不再追（实证存疑已入 KB，机制面闭合即绝源）；
- relay 径不改（v5.5 冻结钦定不动，且现不在跑）。

## 参考

- KB 条目：`docs/工程经验库/条目/headless-outband-spawn-interactive-claim-blind.md`（全时间线+四证锚）
- 前案：`headless-dual-dispatch-path-no-mutex.md`（T-0255/T-0256 无头↔无头面）
- wrapper：`/opt/meta/devin-headless-run.sh` 锁前预检段（T-0256 兼容闸同位）
- 闸链：`/opt/meta/mbus-dispatch.sh` `route_directive_core`（:768 起）/`spawn_rework`（:658 起）

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
