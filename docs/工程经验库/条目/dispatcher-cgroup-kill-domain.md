---
schema_version: 2
id: dispatcher-cgroup-kill-domain
kind: pitfall
risk: high
status: mitigated
keywords: [dispatcher, systemd, cgroup, KillMode, spawn, 无头, 斩杀域]
scope: [ops, mbus-dispatch, headless, systemd]
summary: kill dispatcher 主 pid=在飞无头单全灭——spawn 在 service cgroup 内连坐；setsid 不逃逸 cgroup，正解 systemd-run --scope 或 KillMode
first_seen: 2026-10-30
last_seen: 2026-10-30
evidence_refs:
  - type: log
    ref: ~/devin-dispatch.log 10-30 09:53 三灭窗
  - type: unit
    ref: ~/.config/systemd/user/mbus-dispatch.service
  - type: ticket
    ref: docs/tasks/T-0257-调度面斩杀域根修.md
---

## dispatcher systemd cgroup 斩杀域——杀主 pid=灭全队 spawn

- **实证（10-30 09:53）**：`kill 75707`（mbus-dispatch.service 主 pid）→
  systemd 默认 KillMode=control-group 对 cgroup 全员 SIGTERM→SIGKILL：
  T-0255/0253/0254 三票六进程（wrapper+devin -p）同灭，wrapper 无退出痕。
- **机理**：spawn_run 的 `nohup bash wrapper &` 只免疫 SIGHUP——进程仍在
  dispatcher 的 service cgroup；`Restart=always` 使任何主 pid 死（手动
  kill/crash/systemctl restart）=在飞单陪葬。
- **纠误**：setsid **不**逃逸 cgroup（session 与 cgroup 两域无关）——
  正解=spawn 走 `systemd-run --user --scope`（spawn 自带 scope 单元免疫
  父服务生命周期）或 service KillMode=process/mixed。
- **纪律**：dispatcher 生命周期钦定 `systemctl --user restart|stop
  mbus-dispatch`（restart 同斩杀，根修前亦须避开在飞窗）；裸 kill 主
  pid=事故级。
- **伴生**：headless-supervisor once 键 reboot 内一次性——同票二代 spawn
  殁后不再报死讯（SAID 键首殁已占）；修=dead 键挂 spawn 代际再武装。
- **根修（T-0257 @ 2026-10-30，meta 8952937）**：spawn_run/spawn_audit 改走
  `systemd-run --user --scope`（headless-run|audit-<TID>.scope）——spawn 自带
  scope 免疫父服务生命周期，scope 失败 fail-closed 升 MANAGE 不回退；
  在飞五代树手工迁出 dispatcher cgroup 至 headless-legacy-*.scope 后
  `systemctl --user restart` 实证零殉；supervisor SAID 键挂 spawn 代际
  （日志 spawn 行数=世代号，新一代清键再武装）；生命线纪律成文于
  devin-headless-role.md+service 注释——裸 kill 主 pid=事故级。
