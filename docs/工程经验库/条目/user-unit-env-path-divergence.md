---
schema_version: 2
id: user-unit-env-path-divergence
kind: pitfall
risk: medium
status: verified
keywords: [systemd-user, EnvironmentFile, home径, o-research.env, 部署模板, env路径, 分域切换, 双写漂移]
scope: [deploy/o-research.service, /opt/o-research/env, ~/o-research.env, T-0169 切换]
summary: 用户级 systemd 单元的 EnvironmentFile=%h/o-research.env（home 径）与 deploy/ 模板及 /opt/o-research/env（安装径）双面并存——改安装径 env 不改 home 径=进程照读旧值（T-0169 切 DSN 实挫：research 切后 13min 仍写 SG，产出 5 行提案落错库）；改 env 前必 systemctl --user cat 核 ExecStart/EnvironmentFile 实读路径
first_seen: 2026-10-27
last_verified: 2026-10-27
skills: [生产取证]
evidence_refs:
  - type: server_fact
    ref: "T-0169 切换——/opt/o-research/env 已切 PG-md 后，o-research 仍以 research_rw 持 12 连接写 SG（pg_stat_activity 实拍）；live unit EnvironmentFile=%h/o-research.env；切窗产出提案 5 行（SG 序列 291384-391）落 SG 不落 md，md 新序列撞号二行"
  - type: code_ref
    ref: "deploy/o-research.service 头注附注（T-0165：用户级 systemd 读 %h/o-research.env——附注在案仍被漏读）；systemctl --user cat o-research.service 为实读路径唯一判据"
---

# 用户级 unit 的 env 双径陷阱——改非所读，切换期静默漂移

## 坑形

o-research 的 env 存在两份：deploy 模板记载的安装径 `/opt/o-research/env` 与
live unit 实读的 `%h/o-research.env`（home 径——T-0165 附注明记「部署按用户级
落（差异记档）」）。T-0169 切 DSN 时改了安装径并重启——进程照读 home 径旧值，
**research 臂继续写 SG 13 分钟**，产出 5 行提案落 SG 序列、与 md 新序列撞号
（291384/385 双库各持一义，其中一行被 md 正本前向覆写）。

## 根修

- 改 env 前必 `systemctl --user cat <unit>` 核实 EnvironmentFile 实读路径；
  双径并存的要么删一处、要么在两处同步改。
- 切换类操作的验证锚不能只看「进程 active」——必须看**副作用落点**（本例
  pg_stat_activity 的 usename+client_addr+query 实拍写向；「active」只证活
  不证写对库）。

## 教训

- 部署模板与 live unit 的差异记档若只在头注里，就是给下一个操作者留的雷；
  切换 runbook 的第一步应是「实读路径核对」。
- env 双径期间产生的新数据要以 id 键空间全局对齐收尾（setval 过双库 max），
  不能只回滚配置——配置可回滚，已落行的 id 不会。

## 关联

- T-0165 附注（用户级 systemd 差异记档）
- T-0169 实操事件簿（id 键空间切窗损耗段）
