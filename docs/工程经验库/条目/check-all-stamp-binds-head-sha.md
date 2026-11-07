---
schema_version: 2
id: check-all-stamp-binds-head-sha
kind: pitfall
risk: medium
status: verified
keywords: [check_all, 绿戳, pre-push, stamp, SHA 绑定, 先跑后 commit, 全谱白烧, check-green]
scope: [交付流, pre-push 闸, check_all 时序, worktree]
summary: check_all 绿戳文件名绑定运行时 HEAD SHA——未 commit 先跑=戳落旧 SHA，commit 后必须重跑全谱（本实证白烧一遍 full ~40min）；正序=先 commit 后 check_all 再 push
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: [施工席]
evidence_refs:
  - type: doc
    ref: scripts/check_all.sh:12（STAMP="$GITDIR/check-green-$(git rev-parse HEAD)"——戳名=运行时 HEAD）
  - type: doc
    ref: scripts/git-hooks/pre-push（按被推 SHA 查戳——旧 SHA 戳不豁免新 SHA）
---

## 事实/症状

- 施工席在**未 commit** 的工作树上先跑 `check_all.sh`（full 档 ~40min
  多席并发下更长）——全绿收尾打戳 `check-green-<fa8fe644…>`（领单
  commit 的 SHA）；
- 随后 `git commit` 实现 commit（新 SHA 217f43af）→`git push` 被 pre-push
  闸拒：「无绿戳」——戳绑定旧 SHA，新 SHA 必须重跑全谱，前一遍 full
  白烧。

## 根因或策略解释

- `check_all.sh` 截戳时取 `git rev-parse HEAD`——**未 commit 的改动不在
  任何 SHA 名下**，戳只能落 HEAD（旧 commit）；pre-push 按「被推 SHA」
  查戳是防伪戳/陈戳的对形，两端机械都对，错在调用时序。

## 处置

- 交付正序钦定：**先 commit → 再 check_all（戳即落新 SHA）→ push**；
- 若必须先验证后 commit（如不敢落半成品），用 `CHECK_ALL_FORCE_FULL=1`
  干跑验证内容、心里预留「commit 后还得再跑一遍打戳」的成本——
  两遍跑不了互替；
- docs-only 收尾件（票面回填等）可趁 full 戳在案后再 commit 一次
  docs-only 快径补戳（秒级），避免两遍 full。

## 证据

- check: T-0431 收口实拍（2026-11-07）——第一遍 full 绿戳落 fa8fe644
  （领单 SHA），commit 217f43af 后 push 拒，重跑第二遍 full 才过闸
- result: 两遍 full 全绿、第一遍纯浪费——时序修正后一次到位

## 反证条件

- check: pre-push 若改为「工作树净+HEAD 戳鲜」联合判据则本坑失效
- falsifies_when: 闸逻辑改造后此条目 retired

## 适用边界

- 全体走 pre-push 绿戳闸的席（施工/决策/COS 直办）；仅时序律，
  不涉 check_all 本体语义。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a（时序纪律坑，无量化表——两遍 full 的 token/时耗属
  席位开销非量表域）
- class: n/a
- query: n/a
- threshold: n/a
