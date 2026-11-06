---
schema_version: 2
id: headless-dm-merge-flattened-concurrent-rebase-segmented-push
kind: pitfall
risk: medium
status: verified
keywords: [merge, rebase, master-direct, T-0219, 分段推, 票号闸, 并发席, first-parent]
scope: [机制面, git 流, 部署树, 推送闸]
summary: 决策席 merge 被并发席 pull --rebase 摊平→施工 commit 落 first-parent 撞 master-direct 闸；翻改后归 done 件撞 T-0219 他票占用误判——修=O_ALLOW_NONFF=1 重建 merge+分段推（翻改先推闸免/归 done 后推吃 R 同 blob 豁免）
first_seen: 2026-11-06
last_seen: 2026-11-06
evidence_refs:
  - type: provenance
    ref: "T-0380 合流 2026-11-06——merge c4e14d1c 被并发 pull --rebase 摊平（reflog rebase finish onto daddc4f1），施工提交 d5498cf2/a3268b59 落 first-parent"
  - type: gate
    ref: "pre-push 两红——check_master_direct 施工前缀闸拒+T-0219「他票占用」（done/ 件与远端原票内容异，head_archive_move 豁免不命中）"
  - type: commit
    ref: "修复径 master@2fbf5d31——O_ALLOW_NONFF=1 回退重 merge+票翻先推（区间票件仅 M 闸免）+归 done 后推（R 同 blob 豁免 NOTE 过）"
---

## 部署树 merge 遭并发 pull --rebase 摊平——双闸连环与分段推修径

- **实证**：T-0380 合流 merge commit c4e14d1c 立后被并发席（O-DM
  pane COS 作业中）`pull --rebase` 摊平：rebase 把支上施工提交重放
  成 master first-parent（reflog `rebase finish onto daddc4f1`），
  树内容零变（git diff 前后 tip 空）但 first-parent 链形毁。
- **根因**：推送闸看历史形不看树——①check_master_direct 只许
  [角色:施工] commit 居 merge 第二亲，摊平=§9 违推拒；②T-0219
  head_archive_move 豁免要求归档 blob 与远端原票**同文**——票翻
  完成改了状态行，归 done 与翻改同推必撞「他票占用」误判。
- **教训**：merge 一旦摊平必须重建（闸示 `O_ALLOW_NONFF=1` 回退
  重 merge，勿将就线性史）；决策席推序钦定**分段推**——merge+票
  翻先推（区间内票件仅 M 改→T-0219「无新立票」闸免），归 done 纯
  改名后推（R 同 blob→豁免 NOTE 过）。绿戳按被推 HEAD 计，每段
  各跑 check_all（docs-only 段吃豁免秒过）。
- **收口**：T-0380/T-0379/T-0381 三票分段推落地。并发源=交互席
  pull --rebase 习惯——部署树 master 无 upstream，`git pull` 空转；
  同步用 fetch+人工核，**禁 --rebase 扫平 merge**（摊平即重犯）。
