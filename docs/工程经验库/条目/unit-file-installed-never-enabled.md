---
schema_version: 2
id: unit-file-installed-never-enabled
kind: pitfall
risk: medium
status: verified
keywords: ["systemd", "timer", "enable", "装机漏", "o-unit-watch", "unit_watch_dead", "建制机"]
scope: [ops, systemd, deploy, 建制单元]
summary: systemd 单位文件落位 ~/.config/systemd/user ≠ 启用——deploy_image 不装 timer，手工/部署脚本 cp 落位后忘 `enable --now` = 查件死 DOA；判别链=单位文件在∧零 beat→is-enabled 对拍。
first_seen: 2026-11-21
last_verified: 2026-11-21
skills: []
evidence_refs:
  - type: server_fact
    ref: o-unit-watch.timer/o-unit-watch.service 单位文件在 ~/.config/systemd/user 位（deploy/o-unit-watch.* 仓内正本同）但从未 enable——unit_watch_dead 升件#26706「从无 beat 行」实证；`systemctl --user enable --now` 后首拍 rc=0，ops_headless UNITW beat=alive
  - type: code
    ref: scripts/deploy_image.sh 不含 timer 装机步（grep enable/systemd 零命中）——装机=手工径，enable 靠记忆
---

# 单位文件落位≠启用——装件漏 enable=查件死 DOA

## 坑形

`o-unit-watch.timer`/`o-unit-watch.service` 双件在
`~/.config/systemd/user/` 落位（与仓内 deploy/ 正本一致），
`list-timers` 无影——文件在、systemd 认得、**从不调度**。
配套探针 `unit_watch_dead`（从无 beat 行阈=15min）如实升件，
病灶是「装的面缺一步」不是查件码错。

## 机理

`deploy_image.sh` 只管 o-core 镜像换装，不碰 systemd 装机面；
user unit 落位=手工 cp/symlink 径，`enable` 是独立动作靠人记。
文件在→以为装了→实际 `loaded` 但非 `enabled`，
NEXT/LAST 触发列全空。与 core-hookspath-dead-install-face 同族：
「装的面都不对，后续一切修订空转」。

## 正解

- 装机步三拍：`cp/symlink 落位 → daemon-reload → enable --now`，
  缺第三拍=DOA；装毕必核 `list-timers` 见影+首拍跑过。
- 判别链：单位文件在∧零 beat→`systemctl --user is-enabled <timer>`
  一拍分因（enabled=服务崩/锁卡死；≠enabled=装机漏）。
- 防点：新 unit 件进 deploy/ 目录时同票须载装机三拍判据；
  unit_watch 探针自体（监视者的监视者）已被本坑验证有盖。
