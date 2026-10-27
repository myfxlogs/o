---
schema_version: 2
id: acceptance-hotfix-uncommitted-evaporated
kind: pitfall
risk: high
status: verified
keywords: [验收期直改, 未提交蒸发, 静默回归, 源码真源, worktree]
scope: [process, acceptance, deployment, git]
summary: 验收期 typo 级直改在二进制里实证有效但从未提交→下次从 master 重建静默回归——「二进制证过≠源码有」，验收期改动同 commit 落库才算完
first_seen: 2026-10-27
last_seen: 2026-10-27
evidence_refs:
  - type: ticket
    ref: T-0189
  - type: commit
    ref: t/0189-sigre
---

## 验收期直改「实证过」但未提交的蒸发回归

- **实证**：T-0189 三巡验收期，决策席对 `sigdist/datedFutRe` 做 typo 级
  直改（`-USDT-`→`-[A-Z]+-` 宽判），重建 collector 实证 `basis_ann 拍：2 行`
  后销票。该改动**从未落任何 commit**（git 全史 -S 零命中）；下一票从
  master 重建 collector 时旧正则静默回归→`basis_ann 拍：0 行` 四天。
- **病灶**：验证载体（跑着的二进制）与真源载体（git 源码）分离——
  「二进制证过」在源码面不存在；部署树 D-651 禁码件直写，直改件
  天然无处安放，pull/checkout 即冲。
- **律**：验收期一切码面改动（含 typo 级）必须**当拍**走 worktree→
  commit→merge 落库后才算「修完」；验证二进制只证行为不证在册。
  销票前 `git status --short` 零残留+`git log -S<改片>` 有命中=两件套。
- **同族**：T-0187 跨域 stale 引用、fin_0049 重建吞触发器（T-0190）——
  「改的面」与「查的面」错位是复犯病谱系。
