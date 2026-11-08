---
schema_version: 2
id: systemd-user-unit-path-misses-go-bin
kind: pitfall
risk: medium
status: verified
keywords: [systemd-user, PATH, /usr/local/go/bin, go run, command not found, 127, o-mech, dm_tick, 升件去重掩盖]
scope: [deploy/*.service, scripts/meta, systemd --user 单元, go 依赖子件]
summary: systemd user manager PATH 无 /usr/local/go/bin——单元内 go run/go test 子件 127 command-not-found；症状被升件 24h 去重掩盖呈「偶发物料化径败」假相，交互 shell 复跑不复现
first_seen: 2026-11-08
last_verified: 2026-11-08
skills: [生产取证]
evidence_refs:
  - type: commit
    ref: GW-001 段A mech_tick.sh PATH 尾追补径
  - type: server_fact
    ref: "systemctl --user show-environment PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:...——无 /usr/local/go/bin；go 实体=/usr/local/go/bin/go 与 ~/sdk/go1.27.3/bin/go"
---

# systemd user 单元 PATH 缺 go 径——单元内 go 子件静默 127

## 坑形

systemd `--user` manager 的 PATH 是编译期默认集
（`/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:...`），
不含交互 shell 的 `/usr/local/go/bin`。任何经 user 单元拉起的脚本若调
`go run`/`go test` 子件（本仓 ticketdraft 物料化、dm_suspend 绿窗复跑），
单元内一律 `command not found` rc=127。

症状被**升件同窗去重**二次掩盖：子件败→ops_esc 升件行→24h 内同前缀
只报一行——看账面像「偶发物料化径败」，交互 shell 手跑脚本 PATH 齐全
不复现，定位面极窄（GW-001 段A 排查实证）。

## 根修

- 拉起面脚本头部显式补径且**尾追**（不遮 PATH 桩序——桩测面不破）：
  `case ":$PATH:" in *:/usr/local/go/bin:*) ;; *) PATH="$PATH:/usr/local/go/bin";; esac`
  （mech_tick.sh 同式在案）。
- 新单元/新 go 依赖子件上线前核 `systemctl --user show-environment` 实读
  PATH——与 env 双径坑（user-unit-env-path-divergence）同族不同面：
  那个是 EnvironmentFile 读哪份 env，这是 PATH 里有没有该二进制。

## 教训

- 「交互 shell 能跑」≠「单元内能跑」——PATH 面与 env 面同罪，验证锚
  必须落在真实拉起形态（systemd-run/单测桩 PATH 隔离面都不证单元径）。
- 静默容错+去重报的组合会把持续性环境病灶降维成「偶发告警」——升件
  行频率异常低时反而要怀疑去重键在吞同根因连发。

## 关联

- user-unit-env-path-divergence（同族：user 单元环境面与交互面分叉）
- GW-001 段A（mech_tick.sh 承接 dm_tick 尾段时根修在案）
