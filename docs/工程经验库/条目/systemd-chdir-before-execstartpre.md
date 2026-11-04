---
schema_version: 2
id: systemd-chdir-before-execstartpre
kind: pitfall
risk: high
status: verified
keywords: [systemd, WorkingDirectory, ExecStartPre, CHDIR, 首启失败, oneshot]
scope: [deploy, o-decision-seat, o-regime-analyst, o-econ-verdict, systemd-user]
summary: systemd unit WorkingDirectory 先于 ExecStartPre 对全部 Exec* 生效——scratch 目录不存在时首启 200/CHDIR 整 unit 败，外途建目录后即愈呈间歇迷惑形
first_seen: 2026-11-04
last_verified: 2026-11-04
skills: []
evidence_refs:
  - type: server_fact
    source: postgres
    query: SELECT ts,event,ticket,note FROM ops_event WHERE seat='META-DM'
    observed_at: 2026-11-04T07:43:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

o-decision-seat.service 首启 07:42:31 实败 status=200/CHDIR；
`WorkingDirectory=/var/tmp/o-decision` 在 `ExecStartPre=mkdir -p` 之前生效，
mkdir 永远跑不进；目录经外途创建后 07:43:01 metabus 唤醒即成功。

## 根因或策略解释

systemd 钦定序——CHDIR 对全部 Exec*（含 Pre）生效；WorkingDirectory
不存在=整 unit 首启败，间歇迷惑形（外途建目录后即愈=根因隐）。

## 处置

oneshot+tick 壳内自 cd 件弃 WorkingDirectory 行（dm/verdict/regime 三
unit 同修，tick.sh 内 `cd || exit 1` 已兜）；确需目录者改 tmpfiles.d/
enable 步建非 unit 序兜。同形件排查钦定动作——新建 unit 先核 scratch
目录存在或弃行。

## 证据

- check: systemctl --user start o-decision-seat + journal status=200/CHDIR
- result: 首启败→外途 mkdir 后即愈
- ref: T-0312 第六刀部署实证 / T-0315 票面背景①

## 反证条件

- check: systemd 版本 CHDIR 序文档或新版行为变
- falsifies_when: 若某版 ExecStartPre 先跑则弃行非必要
