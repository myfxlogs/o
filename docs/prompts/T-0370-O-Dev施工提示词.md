# T-0370 O-Dev 施工提示词——无头施工部署树直推缝+收口钩词面误中

你是[角色:施工]。领单授权=业主手动外派（票面领取行标
「业主手动外派」，审结归 COS 不归 DM——collab.md 写面
判据窄口豁免件）。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围）：
   `/opt/o/docs/tasks/T-0370-无头施工部署树直推缝-收口钩词面误中.md`
3. 实证锚先看：
   - `scripts/hooks/pre-push`+`scripts/hooks/post-commit`——
     闸面与钩件现状（误中词面在哪）；
   - `scripts/meta/dev_run.sh`/`dispatch_worker.sh`——施工
     run 持凭径（直推缝的机制入口）；
   - T-0367 谱 `abe3fd1a`（直推 commit 实证）+
     ops_event T-0367 收口行 auto@abe3fd1a（SHA 指错实证）。

## 要点

- 推面缝二择一裁断落实（票面任 1）：pre-push 拒收
  [角色:施工] 前缀 commit（台账白名单豁免与否写明）或
  职卡+提示词钉「部署树零提交」；
- post-commit 词面收紧：「收口——」/「收口」起动作词
  精确形，「收口支SHA」不误中；
- 桩 commit 各形 subject 干跑钩件证误中率零。

## 红线

- 事件词表/队列推导/相位语义零动——纯闸面+钩件精度；
- 钩件 fail-closed：败/解析疑=宁漏报不阻断 commit；
- 撞限流落 WIP+挂起自报再退；不合并不部署不改票面。

## 收口

新 worktree→分支 `t/0370-*`→check_all 绿戳对口 SHA→
report.sh 领单/收口/移交→§8 模板照填（**移交行写 COS
非决策席**）→推送分支即交付。
