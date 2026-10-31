---
schema_version: 2
id: retired-service-enabled-respawns
kind: pitfall
risk: medium
status: mitigated
keywords: [systemd, auto-restart, 退役服务, disable, o-relay, 噪音告警]
scope: [ops, systemd, lifecycle, relay]
summary: 退役 unit 只 stop 不 disable=Restart=always 每 5s 复活刷告警噪音——退役钦定 `disable --now` 双动
first_seen: 2026-10-30
last_seen: 2026-10-30
evidence_refs:
  - type: log
    ref: ~/devin-relay.log 10-30 11:40-11:44 每 5s「relay v5 启动→tmux DM 缺席」循环
  - type: unit
    ref: ~/.config/systemd/user/o-relay.service（loaded activating auto-restart，描述即注『冻结代，M4 退役』）
  - type: handoff
    ref: /opt/meta/docs/管理席交接态.md（『relay v5.5 已停机勿重启』）
---

## 退役 unit 只停不禁=auto-restart 复活——o-relay 5s 噪音环

- **实证（10-30 11:40）**：`o-relay.service`（自描述『冻结代，M4 退役』）
  在 `loaded activating auto-restart` 态——relay v5 早已钦定停机，但 unit
  仍 enabled+Restart=always，每 5s 拉起一次打两行日志
  （v5 启动计数行+『tmux DM 缺席——停环待人工』），日志噪音无限续。
- **机理**：`systemctl stop` 只刹当前实例——Restart=always 让 systemd
  按间隔拉起；**停机≠退役**。前任「已停机勿重启」纪律停在口头层，
  机面 enabled 未除，偶发触发（人工误 start/依赖拉起/linger 重启）即复活。
- **正解**：退役处置=`systemctl --user disable --now <unit>` 双动
  （摘 wants 链+停当前）——11:44 管理席执行后日志清净。
  同理适用一切「钦定退役」的 systemd 面：纪律成文之外，机面 disable 才算数。
- **查法**：`systemctl --user list-units --type=service` 找
  `activating auto-restart` 态——反复拉起的服务一律现形。
