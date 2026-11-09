---
schema_version: 2
id: git-filemode-exec-bit-lost-rc126
kind: pitfall
risk: medium
status: observed
keywords: ["exec bit", "100644", "100755", "rc126", "权限拒", "git ls-files -s", "file mode", "调度直调"]
scope: [git文件mode, .sh脚本, systemd/cron调度面]
summary: .sh 入库丢执行位（git 记 100644）→调度面直调 rc=126 静默拒跑；入库核 mode+调度走 bash <script> 免位依赖；rc126=权限拒与 rc127=找不到分治。
first_seen: 2026-11-08
last_verified: 2026-11-08
skills: []
evidence_refs:
  - type: doc
    ref: docs/工程经验库/待归纳.md（staging 原档——纪元3归纲转正）
---

## 事实/症状

dm_autoapprove.sh git 记 100644 缺执行位——调度面直调 rc=126 机批闸径败，候选批准面滞留（ops_event#3566，2026-11-08；根因落行#3632）。

## 根因或策略解释

文件创建/amend 流程中执行位未保留——git 只记 mode 位，100644 入库后 `./script.sh` 直调=permission denied。手跑 `bash script.sh` 不受影响故开发期不可见，调度面直调才炸。

## 处置

- .sh 入库必保 100755——`chmod +x` 后 `git ls-files -s` 核 mode（应见 100755）
- 调度面直调脚本前 `test -x` 或走 `bash <script>` 免位依赖
- 排障分治：rc=126=权限拒（查 file mode），rc=127=找不到（查路径）

## 证据

- check: staging 原档 ops_event#3566/#3632 实案
- result: git 记 100644→直调 rc126
- ref: 待归纳 staging 原文（本条目转正件）

## 反证条件

- falsizes_when: 仓规/钩强制 .sh 入库 mode=100755 检查（入库闸拦截）

## 适用边界

- 一切 git 跟踪的可执行脚本（scripts/、hooks/、wt 内件、镜像内脚本面）；bash 显式调用径不受影响但属侥幸非设计

## 伴生量表（T-0332）

- gauge_name: n/a
- class: 缺席
- query: n/a
- threshold: n/a（豁免：入库卫生件——判据=git ls-files -s 人工核/钩可补闸）
