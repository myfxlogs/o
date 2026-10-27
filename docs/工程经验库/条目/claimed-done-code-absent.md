---
schema_version: 2
id: claimed-done-code-absent
kind: pitfall
risk: high
status: verified
keywords: [空头修, 回填失真, diff为证, 验收对拍]
scope: [review, acceptance, ticket-backfill]
summary: 返修commit只回填写「已改」码面零动——验收永远以diff/对拍为证，票面文字非证
first_seen: 2026-10-27
last_seen: 2026-10-27
evidence_refs:
  - type: ticket
    ref: docs/tasks/T-0182-TRX采集三流间歇断供.md
  - type: commit
    ref: 35acd8c
---

## 空头修——回填写「已改」码面零动（验收唯一证=diff）

- **实证**：T-0182 返修 commit 35acd8c 回填写「四处接线同一刀」+附
  curl 对拍，但 `git show` 全 diff=纯文档 10 行——tronscanBase 三处
  死径（api.tronscan.org 死域+/api/api/ 双段）码面**原样未动**。
  「已修」声明与实物完全脱节。
- **病灶型**：比改错更劣——改错尚有迹可验，空头修用文字把真病灶
  再盖一层（若验收信票面文字不核 diff，死径以「已修」名义永存）。
- **判据钉死**：验收唯一证=diff+生产对拍；票面回填/施工自述/附带
  「已验证」文字**均非证据**。返修件第一查=commit 是否触到返修段
  钦定的文件（零触=空头直接退回不读回填）。
- **同族**：修而未合=未修（port-wiring-drift 形态三）的姊妹形态——
  一个滞留分支一个滞留纸面，同属「修≠落地」家族。
