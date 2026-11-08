---
schema_version: 2
id: orphan-pid-reparent-user-systemd-subreaper
kind: pitfall
risk: medium
status: verified
keywords: [ppid=1, orphan, reparent, systemd --user, subreaper, fuser, 锁, 孤儿锁, wt_gc, check_all]
scope: [scripts/meta, bash, systemd --user, 进程树]
summary: 「父死孤儿」判据写成 ppid=1 在 user systemd subreaper 机器上恒假漏杀——本机 systemd --user(1275) 是 subreaper，o-worker/tmux 孵化的孤儿 reparent 到它而非 pid 1。修法：父死判据两形并查（ppid=1 ∨ 父 comm=systemd），且须实证孤儿真实 reparent 落点再定判据
first_seen: 2026-11-08
last_verified: 2026-11-08
skills: [施工席]
evidence_refs:
  - type: doc
    ref: scripts/meta/wt_gc.sh
  - type: doc
    ref: scripts/meta/tests/disk_hygiene_test.sh
  - type: server_fact
    ref: 11-08 亲证——kill check_all wrapper 后其 go test 子树 ppid=1275（user systemd）非 1，ps -o comm= -p 1275=systemd；字面 ppid=1 判据漏杀实证
  - type: doc
    ref: docs/tasks/done/T-0466-磁盘保洁-deploy构建产物无出径致根分区100%事故.md
repro_or_symptom: wt_gc 孤儿锁扫杀按「持 fd 者 ppid=1」判父死，实测杀 holder 的 wrapper 后其子进程 ppid=1275（comm=systemd）非 1——判据恒假，孤儿锁持者漏杀，/tmp/o-check_all-*.lock 长期占锁致新跑傻等 1800s 超时
root_cause: systemd --user 实例以 PR_SET_CHILD_SUBREAPER 成为本子树 reaper——其域内任何进程死后孤儿 reparent 到该 user systemd 而非 PID 1（本机 pid 1275）。o-worker transient unit/tmux 孵化链皆在 user systemd 域内，「孤儿→ppid 1」经典假设不成立
fix_and_prevention: ①父死判据两形并查：`pp=$(ps -o ppid= -p $pid)`，`[ "$pp" = 1 ] || [ "$(ps -o comm= -p $pp)" = systemd ]`——合法律（ppid=跑席 pid 非 reaper）不误伤；②凡写「父死/孤儿」类判据前先亲跑实证 reparent 落点（kill 中间父→ps 观孤儿子 ppid），不同 init/reaper 域落点不同；③桩测 spawn 孤儿须让中间父先死再断言，且断言对拍真实 reaper 形态
notes: 与 wt-gc-dead-gate-subshell-counters 同族（「会动的闸」未实证=死件嫌疑）——恒 false 的判据比没判据更毒。桌面/容器 init 域（tini/dumb-init）reaper 落点又不同，跨机移植须重实证
---

## 现象

`wt_gc.sh` 孤儿锁扫杀判「持 fd 者 `ppid=1`」收尸，桩测 fixture（`setsid sh -c 'exec 9>锁; sleep 300'`，杀其父）在案漏杀——实测孤儿 ppid=1275（comm=systemd）非 1。

## 根因或策略解释

systemd --user 实例在本机是 subreaper（pid 1275）：o-worker systemd-run transient unit 与 tmux pane 孵化的进程树死后，孤儿 reparent 到 user systemd 而非 PID 1。Unix 经典假设「孤儿必 ppid=1」仅在 init 直接域成立；subreaper 域内判据恒假=漏杀死件。

## 处置

`orph_ppid()` 两形并查：`ppid=1 ∨ 父 comm=systemd`；锁文件绝不删（防双 inode 锁破）；孤儿杀子孙+本体+ops_event 报行，dry 只检不杀。桩测断言对拍 reaper 真实落点。

## 证据

- check: `kill <check_all wrapper pid>` 后 `ps -o ppid=,comm= -p <go test pid>` 与 `ps -o comm= -p $(ps -o ppid= -p <pid>)`
- result: 孤儿 ppid=1275，其父 comm=systemd（非 pid 1）；orph_ppid 两形判据命中、字面 ppid=1 判据漏
- ref: T-0466 收口 commit（t/0466-disk-hygiene）

## 反证条件

- check: 换 init/reaper 域（无 user systemd 的机器/容器 tini 域）复跑 fixture
- falsifies_when: 若某环境孤儿 reparent 落点既非 pid 1 也非 systemd（如 tini/kata 自设 reaper），判据需再加形——故判据应随部署域实证重校，不钦定两形完备

## 适用边界

- 本机 systemd --user subreaper 域（o-worker/tmux 孵化链全在内）；
- fuser 扫 `/tmp/o-check_all-*.lock` 持 fd 集场景；
- 合法律 ppid=跑席 pid（非 reaper）不误伤在案，新增 reaper 型父进程（如自托管 supervisor）需复核。
