---
id: landqueue-residual-merge-mandate
title: 单写者机械合上线后——旧「部署树 merge」契约行漏网=分叉源复发
kind: pit
date: 2026-11-17
source: 工单#22949（deploy_diverged 三发+落地链死锁，P1）
---

# 契约面改语义须全仓扫同径词面，不只改主流程文

**坑形**：land_queue（工单#22577/D-0048③）把「复审过支→master」
机械合收归单写者后，runbook「要码修」径仍钦定「部署树 merge
--no-ff」——本地 merge 点不入 origin（mirror_push 只推支尖 sha），
本地 HEAD 恒在 origin/master 外：land_queue 基线闸每拍让位
「master≠origin」+deploy_watch 让位盖戳流=落地链死锁，reset 清障
同族复发四次+（#22937 是补丁非修）。堵盘烧修/复审支/一切后支。

**判别**：新机制上线后旧路径的**契约行**（runbook/戒律/闸引导文）
不随机制同批裁——机制对了，指令还在教人走旧径。分叉源不在码在文。

**修法**：凡「X 归某单写者」类架构收编，落地同 commit 必须 grep
全仓同径动词面（`merge --no-ff`/`pull/merge/deploy` 权限行/闸 FAIL
引导文/audit 径简写），逐处改 pull-only 或指向新单写者——本案四处：
cognition/runbook 要码修径+审计径、worker_runbook 戒律行、collab.md
§3、check_deploy_commit_seat 引导文。

**推广**：收编类改动的验收面=「旧径指令在仓内零残留」可机检
（grep 词面），不是「新机制能跑」——机制跑而旧指令活=双轨必撞。

**关联**：land_queue 单写者（scripts/land_queue.sh）、deploy_watch
ff-only 对齐、mirror_push 推已绿 sha 非 HEAD。

