---
schema_version: 2
id: listen-pool-conn-contamination
kind: pitfall
risk: critical
status: verified
keywords: [LISTEN 池化污染, ev_eval 高频化, 自激洪泛, WaitForNotification, pgxpool, fin_proposal 洪泛]
scope: [internal/platform/events/, 所有 ev_* 事件环, 提案/评估写径]
summary: pgxpool 借连接跑 LISTEN 是会话级常驻且还池不清理——下任借用者继承频道残留；ev_eval 扩 UPDATE 后成高频频道，污染连接让 funding 扫拍被自激唤醒：扫拍写 eval→eval 唤醒扫拍，提案表 5k 行/分灌穿实证
first_seen: 2026-10-25
last_verified: 2026-10-25
skills: [生产取证]
evidence_refs:
  - type: production_fact
    ref: "2026-10-25 SG：fin_proposal 10min +39,739 行（181k→244k）；禁 ev_funding/ev_eval 触发器均不止血=唤醒源在连接残留频道；fin_proposal BEFORE INSERT 去重闸后洪泛 5k→0 行/分实证"
---

## 病灶

`events.ListenLoop` 从 pgxpool 借连接执行 `LISTEN <ch>`——LISTEN 是
**会话级常驻**，还池不注销。连接池复用拓扑下，一根曾被 lifecycle
（ev_eval 等）听过的连接被 funding 环借到：funding 只想听 ev_funding，
实际收到借来的全部频道。

T-0126 前 ev_eval 只发 INSERT（罕发），扩成 INSERT OR UPDATE 后变成
**每帧必发的高频频道**——污染面从不显著跃升为自激引擎：
扫拍→写 eval→ev_eval 唤醒同一连接上的扫拍→再扫……死循环。

## 判据

- **写事件环**：任何用池连接 LISTEN 的代码，还池前必须 `UNLISTEN *`
  （defer 里钉死）；或干脆用专用非池连接。
- **写触发器**：高频更新表上的 NOTIFY 触发器=唤醒广播面——评估频道
  频率量级×环响应成本，超阈即自激风险。
- **诊断**：禁触发器不止血的 NOTIFY 驱动洪泛=查连接级残留频道；
  洪泛根治闸=在写入入口做幂等去重（BEFORE INSERT gate），
  比追溯唤醒源可靠。

## 根因

LISTEN/NOTIFY 的会话语义 vs 连接池复用语义的错配——两层各自正确
（PG 会话常驻、池复用透明），组合后频道归属不可见。T-0126 合理的
UPDATE 触发器扩展只是把它从低频引爆成高频。
