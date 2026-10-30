---
schema_version: 2
id: ts-watermark-vs-stamped-ts-order
kind: pitfall
risk: high
status: verified
keywords: [projector, watermark, md_signal, carry_diff, mt5_swap, 血统ts, 镜像失明, wm跳读]
scope: [projector, mirror, signal-chain, SG]
summary: ts 水印假设「stamped ts≈commit 序」——血统回戳/迟提交行出生即在 wm 后→永久静默跳读零报错；绿信号行幸存只因 ts≈实时
first_seen: 2026-10-30
last_seen: 2026-10-30
evidence_refs:
  - type: ticket
    ref: docs/tasks/T-0272-信号道水印失明根修.md
  - type: decision-context
    ref: docs/FIX02-收口.md（fix/mt5-swap-outage——"僵死"判词被本坑修正）
---

## 镜像水印按行戳推进=迟可见行永久跳读（stamped-ts≠commit-序 全类病灶）

- **病灶**：projector `ev_signal` 道 `SELECT … WHERE ts>wm` 把行戳当提交序。
  carry_diff 按 T-0164 血统闸写 `ts=leastTs(fundingTs,swapTs)`（落后达 8h）、
  funding_view 每行自盖观测戳——**行落地瞬间 ts 已在 wm（≈xvenue 实时沿）之后**，
  事件径查不出、活性对账 `max(wm,now-10min)` 捞不回：永久静默丢失，零日志零报错。
- **假象谱**：SG「断供/冻结」逐 kind 各死一拍（mt5_swap 01:13、basis_ann 02:55、
  funding_view 17:30），xvenue_spread 独活——不是进程僵死，是各行戳与 wm 沿的
  相对位置决定生死。进程被 SIGQUIT 杀出的寄存器 dump 被误判「僵死 13.5h」
  （FIX-02 判词修正案：正常路径零日志≠冻结）。
- **serve 谓词第二伤**：signalGlobalKinds 只含 carry_diff/xvenue_spread——
  mt5_swap/basis_ann 在 serve 模式下**设计性不镜**（SG 侧 xvenuepair swap_snapshot、
  ops_face 蒸馏行实需）。谓词是按体量裁的闸，小表别裁。
- **判则**：凡「行戳由业务/血统语义回填」的表（latest-value upsert 类），镜像/增量
  同步**禁按行戳做 watermark**——要么 arrival 列（recv_ts/wts 径，md_funding
  T-0169 已先吃此修）要么小表全量谓词扫。
- **排查捷足**：SG vs 真源双端 `max(ts)` 对拍 + 重启 replay 落地=水印跳读实锤
  （replay wm 归零全拉，跳读行一次性补齐——2026-10-30 23:15 实证 carry_diff
  17:11→22:35、funding_view 17:30→23:14）。
