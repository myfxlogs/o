---
schema_version: 2
id: sg-pool-halfopen-wedge-silent-stall
kind: pitfall
risk: high
status: verified
keywords: [projector, SG镜像, ssh隧道, 半开连接, ctx超时, 静默卡死, md_tick_scan, mirror stall, pgxpool]
scope: [projector, pgstore, tunnel, o-projector]
summary: PG 池借出连接经 ssh 隧道半开静默死——查询永等无 ctx 界，共享池的全部 goroutine 冻结，进程仍活零日志零告警；唯一外部征兆=下游新鲜度探针（dep_blind/repair.feeds）报警
first_seen: 2026-11-25
last_verified: 2026-11-25
skills: []
evidence_refs:
  - type: ticket
    ref: O-37543
  - type: code
    ref: internal/projector/projector.go
---

## 现象

o-projector 进程活着、journal 零错误，但 SG 侧 `md_tick_scan`（及全部
桥镜像表）`max(ts)` 自 ~03:31 起冻结约 4 小时，直到 07:33 deploy-oci
重启后自愈。PG-md 侧同表全程新鲜——断点不在采集臂，在投影臂→SG 的
写腿。同窗口 collector okxscan 拍 278 连拍零败、PG-md `recv_ts`
~25s 鲜——三源交叉排除「ScanPoller 停写/REST 败/goroutine 卡死/
二进制回归」（ace6bd60→43d4ce5 diff 扫描路径零触）。

## 根因

projector 的 SG 池（`projector_rw@127.0.0.1:15433`，ssh -L 隧道转发
到 SG PG）借出连接遭遇**半开 TCP**：隧道/对端静默断链后，本端
socket 不感知（TCP keepalive 默认数小时才判死），pgx 查询在该连接
上**无限期阻塞**——调用方 ctx 是进程生命周期 ctx（永不取消），
无任何超时界。后果链：

1. 冻结连接上的查询永久 hang → 占用的 goroutine 不归。
2. 连接池逐步被挂起查询占尽 → `Acquire` 也阻塞 → beatLoop
   （roster.Beat+mirrorStaleSince）、livenessLoop（Reconcile）、
   mdWake/sgWake 事件 fn **全部静默冻结**。
3. 进程仍活 → systemd 不重启；所有日志面静默 → 零告警。
4. 连「镜像静默哨」（mirrorStaleSince 本应报警 projector_silent）
   也因同池冻结而失声——**检测器与被检测对象共用故障域**。

唯一发现路径=下游消费面的外部新鲜度探针（dep_blind 10min 窗、
后续 repair.feeds 补册）。

## 处置（O-37543 落地形态）

- **ctx 界律**：projector 一切 SG 面长操作挂 `context.WithTimeout`
  ——beat/哨兵单操作 `sgOpTimeout=90s`、对账（boot 全量+liveness）
  `reconcileOpTimeout=20min`（boot 全量实测 ~3.6min，2×livenessEvery
  余量）。超时→error→既有 `errc` 岗终律→`log.Fatalf`→systemd
  `Restart=` 拉起→新连接池自愈。wedge 从「静默 4h」压到「≤90s 检出
  +重启收敛」。
- **界值纪律**：超时必须宽于正常时延尾部（beat=单行 upsert 毫秒级；
  对账分钟级），否则健康慢查询被误杀成重启环。
- **repair.feeds 补册**：`md_tick_scan` 入册（unit=o-collecto，
  dead_sec=600）——注意它看的是 **PG-md 正本侧**新鲜度，只覆盖
  「采集臂停写」场景；镜像腿 wedge 由 ctx 界+errc 自愈覆盖，两案
  不同故障域不可互替。
- **同类排查指引**：凡「进程活+日志静默+下游数据冻结」三联，先查
  该进程对外部依赖（PG/HTTP/gRPC）的连接是否半开——pgx/pgxpool
  默认不设 statement_timeout，ctx 不取消=查询永等；修法不是调池
  参数（HealthCheckPeriod 的 Ping 同样会 hang 在半开连接上），
  而是调用点 ctx 界。

## 验收

- 桩测/对拍：projector.go diff 仅 ctx 包装+两常量，语义零变。
- 运行时：SG `md_tick_scan` `max(ts)` recv_age ~25s 双侧新鲜在案。
- repair.feeds generation 5484 三条目在 SG+md 副本双侧核实。
