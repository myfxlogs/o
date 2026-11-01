---
schema_version: 2
id: headless-outband-spawn-interactive-claim-blind
kind: pitfall
risk: high
status: observed
keywords: [无头双发, 带外spawn, systemd-run, 交互席盲区, 已领未推, claim锁窗, devin-headless-run, mbus-dispatch, 让位, T-0283, T-0256]
scope: [meta调度面, /opt/meta/devin-headless-run.sh, /opt/meta/mbus-dispatch.sh, /opt/o/docs/collab.md §3]
summary: 同票无头副本由「带外手起 systemd-run」spawn（绕 dispatcher 全闸）落入交互席「已领未推」窗——远端无支=判未领假阴性；wrapper 只防无头不防交互席，靠 worktree 在案+真席途中推支才让位 rc=0 零写，白烧 1.28M tokens
first_seen: 2026-11-01
last_verified: 2026-11-01
skills: []
evidence_refs:
  - type: doc
    ref: ~/devin-headless-T-0283.log（07:48:04 spawn→07:52:52 让位声明全文+用量 tokens in=1280951）
  - type: doc
    ref: ~/devin-dispatch.log 07:36:53「gate T-0283：票面状态『已批』不在派发集——不派」（闸已拒，spawn 非 dispatcher 径）
  - type: server_fact
    ref: journalctl --user 07:48:04「Started headless-run-t0283.scope - /usr/bin/bash /opt/meta/devin-headless-run.sh T-0283 docs/tasks/…」（scope 名小写无横+argv 相对径=非 spawn_scope 产形：dispatch 件全为 headless-run-T-XXXX+description「headless $u」+绝对径）
  - type: doc
    ref: /opt/meta/mail/.spawned/（无 T-0283 锚——claim_spawn 未走，带外实证）
  - type: doc
    ref: /opt/meta/mail/Dev/acked/20261101-073653-DM-3459.mb（交互席 07:37:23 ack 即领单）
  - type: commit
    ref: b245dc24（交互席 07:39 施工 commit；t/0283-pipe-exit 推远端=07:51——锁单凭据晚 spawn 3min 才成立）
---

## 事实/症状

2026-11-01 时间线（全锚在案）：

- 07:36:53 dispatcher route_directive 闸**已拒** T-0283（票面「已批」不在派发集）——合法径零 spawn；
- 07:37:23 Dev 交互席 ack DM-3459 领单施工，07:39 落 b245dc24；
- **07:48:04 `headless-run-t0283.scope` 起 `devin-headless-run.sh T-0283`（pid 772918）**——scope 名 `t0283` 小写无横、argv 相对径 `docs/tasks/…`、dispatch log 无 spawn 行、`.spawned` 无锚、无 `--description="headless $u"` 前缀——**四证合一=带外手起 systemd-run，绕票态/in_flight/claim_spawn 全闸**；
- 副本入场三相核实初判「远端无 t/0283-* 支=本席唯一在飞」——**真席已领未推（推支 07:51 晚入场 3min），锁单凭据窗内空转**；赖 `/opt/o-wt/t0283` 工作树+未推 commit 在案起疑，二次核实撞见 devin acp 席+deploy pid 793958+远端支已推→按让位协议 07:52:52 rc=0 零写退出；
- 代价：tokens in=1,280,951（SWE-2 High）白烧+监理告警+MANAGE 核查面；若真席推支再晚数分钟/副本判「前席已死」接管在树工作=真双写。

## 根因或策略解释

三缺叠加：

1. **带外 spawn 无闸**——dispatcher 三闸（票态/in_flight/claim_spawn）只拦自家 spawn_run/spawn_rework；任何席手跑 `systemd-run -- bash devin-headless-run.sh` 直通 wrapper。wrapper 自身只防「同票无头」（flock+pgrep argv 锚，T-0256），对交互席（`devin acp`/tmux 席）零感知、对「已领未推」零感知；
2. **锁单凭据=推支时点证据非领单时点**——collab §3「远端 t/NNNN-* 支存在=锁」在「已 ack 领单→未推支」窗内恒假阴性（本窗实证 14min：07:37 领→07:51 推）；副本以此判「未领」属判据盲区非判据失效；
3. **互斥象限不全**——T-0256 flock=无头↔无头；交互席不入锁不持锁，交互↔无头向只靠 agent 三相核实自觉兜底（自觉是行为不是机制）。

## 处置

立 **T-0285**（已批）钦定三修：

- wrapper 锁前预检扩「已被领」四据——远端支（ls-remote/已抓 refs）∨ `/opt/o-wt/t<NNNN>*` 工作树∨交互席进程相（devin acp/tmux argv 票锚）∨Dev 箱 directive 已 ack——任一命中=让位声明+MANAGE 报+rc=0，零 devin 调用；
- dispatcher spawn 前同据预检（两径判据同源，防漂移）；
- collab §3 钉「领单即推 claim 支先于动工」收窄未推窗；role.md 增「带外手起 wrapper=违例+救火须过同一预检」+wrapper spawn 行记 caller 父链使带外可归因。

## 证据

- check: `journalctl --user --since '07:47' --until '07:49'` vs `grep 'T-0283' ~/devin-dispatch.log`
- result: scope Started 行在案 vs dispatch 仅 07:36:53 闸拒行——spawn 非调度器径实锤
- check: 副本 log 让位声明三相表（进程相=devin acp pid 312342+deploy pid 793958；支相=t/0283-pipe-exit @b245dc24 已推）
- result: rc=0 退出、零支零 commit 零 fin_* 行、用量 in=1280951
- check: 副本入场初核「Remote has no t/0283-* branch——本席即唯一在飞席」
- result: 已领未推窗内远端支判据假阴性实证（推支 07:51 后二次核实才翻判）

## 反证条件

- check: 找到 `headless-run-t0283` scope 的脚本化调用行（crontab/他脚本/systemd unit）
- falsifies_when: 存在=「带外手起」归因改判为「隐藏脚本径」，闸点改落该径
- check: 已领未推窗内副本的 worktree/进程相核实亦不命中（如前席未建 worktree 即动工）
- falsifies_when: 窗比实测更宽——T-0285 预检须加「Dev 箱已 ack directive」一据方能闭合

## 适用边界

单机闸面（同 T-0256——/tmp 锁与本地证据不跨机）；**spawn 席位归因存疑**——时间窗内可证活跃者仅交互席面（Dev 在干同票本体排除、MANAGE/Claude 管理域席存疑），MANAGE 原报「dm-wake-075123 所起」与时间线不符（spawn 07:48:04 < dm-wake-075123 点火 07:51:23）已正；audit wrapper 闸面沿用 T-0256 钦定界不扩；驳回返修豁免逻辑不动。
