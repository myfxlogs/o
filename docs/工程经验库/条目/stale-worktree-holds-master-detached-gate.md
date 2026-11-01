---
schema_version: 2
id: stale-worktree-holds-master-detached-gate
kind: pitfall
risk: medium
status: verified
keywords: [worktree, master分支占用, detached HEAD, D-651提交闸, 陈worktree]
scope: [git, worktree, deploy-tree, gate]
summary: 陈 worktree checkout master 支→主树/部署树 detached；D-651 闸按「检出分支=master」判→detached 即拒 commit，且 branch -f 被占用拒
first_seen: 2026-10-31
last_seen: 2026-10-31
evidence_refs:
  - type: commit
    ref: 0a3d04ea
  - type: provenance
    ref: 清孤会话 T-0272/T-0273 判词落票受阻实录
---
---

## 陈 worktree 占 master 支→部署树 detached→提交闸/引用更新双堵

- **实证**（2026-10-31）：`/opt/o-wt/t0160-fixup` 陈 worktree checkout=master
  支+大面积脏树积压，致 `/opt/o` 部署树只能 detached HEAD 工作——
  ①D-651 闸「部署树检出分支非 master 禁 commit」误伤（HEAD 在 master
  内容上但 ref 悬空即拒）；②`git branch -f master origin/master` 报
  「cannot force update the branch used by worktree」；③`git pull`
  在 detached 下无 upstream 语义（输出 usage 行）。
- **解法**：占支 worktree `git checkout --detach`（非破坏——文件/提交
  原样，仅释分支占用）→ `git branch -f master <sha>` 或 `git checkout
  master` 各归位。脏树权责人自处，勿清。
- **通律**：worktree 长期占共享分支（master/release）=协作面定时雷；
  陈 worktree 盘点应纳 session 收尾面（同 .spawned 代际面）。
