---
schema_version: 2
id: git-push-sh-bare-run-worktree-head-master
kind: pitfall
risk: high
status: verified
keywords: [git_push.sh, worktree, HEAD:master, 推送, pre-push, non-fast-forward, 施工席, 断链自报]
scope: [scripts/git_push.sh, git worktree, t/* 施工分支, 推送径]
summary: worktree 内裸跑 scripts/git_push.sh（缺省 refspec=HEAD:master）会向 master 推分支 commits——非快进侥幸拒收；分支与 origin/master 齐平时将快进直推 master 绕过决策席合并权；worktree 推送必须显式 `git push origin HEAD:<t分支>`
first_seen: 2026-11-05
last_verified: 2026-11-05
skills: []
evidence_refs:
  - type: commit
    ref: 755268ee
  - type: doc
    ref: scripts/git_push.sh:8（REFSPEC 缺省 HEAD:master——部署树件）
  - type: doc
    ref: docs/collab.md §4（验收通过由决策角色合回 master——施工无 master 推送权）
---

## 事实/症状

T-0352 施工 worktree（`t/0352-venue-fail-gate`，领先 origin/master 2 commit）收口推送时，pre-commit WARN 提示「用 scripts/git_push.sh 带重试推送」，照办裸跑 `bash scripts/git_push.sh`——脚本缺省 `REFSPEC=HEAD:master`（scripts/git_push.sh:8），三次尝试均 `git push origin HEAD:master`，因分支 behind 9 被 non-fast-forward 拒收。本意是推 t/ 分支，实际瞄准了 master。

## 根因或策略解释

`git_push.sh` 是 D-588 部署树推送包装器（/opt/o 常驻 master 面），缺省 refspec 写死 `HEAD:master`——设计前提=调用处 HEAD 即 master。worktree 里 HEAD 是 `t/NNNN-*` 施工分支，同一缺省值语义全变：不是「推当前分支」而是「把分支 commits 推向 master」。pre-commit 的 WARN 文案不分树/支语境，worktree 内同样触发，诱导照搬部署树用法。本次未酿祸纯靠分支 behind origin/master 被 git 非快进拒收兜底——若分支与远端齐平，本票 commits 将快进直落 master，绕过决策席验收合并权（collab §4：验收通过由决策角色合回 master）。

## 处置

- worktree/施工分支推送一律显式 refspec：`git push origin HEAD:refs/heads/t/NNNN-slug`（本票实证 997bbd93..755268ee 放行）。
- pre-push 钩的绿戳/票号对账只验 commits 内容不验目标分支——挡不住「绿戳分支推 master」这一形态，纪律在调用侧。
- 根治候选（归决策裁）：git_push.sh 加「非 master 分支拒跑缺省 refspec」断言，或 WARN 文案按 `git branch --show-current` 分情形提示。

## 证据

- check: `git ls-remote origin t/0352-venue-fail-gate` → `755268ee…`（显式 refspec 推送成功）；`bash scripts/git_push.sh` 输出 `git push origin HEAD:master 连续 3 次失败`（非快进拒收三连）。
- result: 裸跑三次均瞄 master 被拒；显式 `HEAD:refs/heads/t/0352-venue-fail-gate` 一次放行。
- ref: commit 755268ee；scripts/git_push.sh:3,8。

## 反证条件

- check: 核 scripts/git_push.sh 现行缺省 refspec——若已改为跟随当前分支或加非 master 断言，本条目失效退役。
- falsifies_when: 脚本缺省行为不再是 `HEAD:master`，或 worktree 内裸跑实测推到 t/ 分支而非 master。

## 适用边界

仅 `scripts/git_push.sh` 无参裸跑场景；显式传 refspec（`git_push.sh origin HEAD:t/x`）不受影响。部署树 /opt/o（HEAD=master）用法不受影响——缺省即正解。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a（豁免由：流程机器/工具链用法坑非产线数据面——无适配 gauges.tsv 表；
  监测面=pre-push 钩+远端分支实查人工径）
