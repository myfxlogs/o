# O-Dev 施工提示词 —— T-0350 管道段表达根修

> 派工依据：业主令（2026-11-05 01:0x）——「那就是表达得不对」。提示词即领单
> 授权，COS 代业主起草自审过闸。票已 `已批` 在案，票面即范围，勿再诊断病灶。

## 你的身份与通用纪律

- 你是 `[角色:施工]`，一 session 一角。先读 `/opt/o/AGENTS.md`、
  `/opt/o/docs/collab.md`、`/opt/o/skills/施工席/SKILL.md`。
- 仓库 `/opt/o`；实现面**必须开 worktree**（master 仅容 docs/**）。
  worktree 起后先跑 `bash scripts/provision_worktree.sh`。
- 提交格式 `[角色:施工][T-0350] 摘要`；`bash scripts/check_all.sh` 绿才可
  收口移交（web 构建含——dist 须在）。
- 里程碑 `~/bin/report.sh 领单|收口|移交 T-0350 "备注"`。
- 收口即 `移交` 停手等审计，不 merge 不 push 不部署。限流=挂起退出。

## 任务

票面：`/opt/o/docs/tasks/T-0350-管道段表达根修-需求驱动段龄改态标.md`

已替你踩过的点：

- **段定义面**=`cmd/core/flow_pulse.go` 头段表（35-41 行：`collect/bridge/
  eval/proposal/exec/report/scout` 七段，末列 bool=需求驱动标记面已存
  ——`proposal`/`exec` 标 `true`，先读明白这列现有消费径再动）。
- **前端渲面**：找渲 `采集/投影/评估/提案/执行/报表/侦察` 注行的组件
  （Cockpit.tsx 或专卡组件——读 props 里 stage lag 如何成文）。
- **生产实据可对拍**（SG 直查或经隧道 FIN_PG_DSN）：
  `fin_opp_eval` 近 24h 93 行全 rejected、`evidence->>'reason'` 形如
  `net_apr -0.3346 < 阈 0.0600`——态标拒因短标从此出；
  `fin_proposal` 零 armed（7d 67 件全 fin_retired）；
  `fin_cashflow` 末行 kind=interest（被动类）——「真执行类」白名单/
  排除集判据票面留你裁，以 `kind` 实际枚举实查为准（`\d` 或 sample）。
- **与 T-0349 的界**：那票修①存活卡 META 行；本票修管道七段卡。
  若 O-Dev 你正干 T-0349，两票分支独立 commit 链不混。

## 停止条件

- 需动 fin_* 数据面或加表 → 超票面，停，回报。
- 「真执行类」cashflow kind 枚举实查与票面假设冲突 → 按实查改判据
  并在收口备注记取舍，不需停。

## 自审记录（提示词五查，起草方填）

| # | 查项 | 发现 | 处置 |
|---|---|---|---|
| 1 | 依据链 | 业主令原文+票已批在案 | 提示词首段即授权依据 |
| 2 | 指代歧义 | 「态标」文案自由度 | 判据钉死文案样式裁量归施工 |
| 3 | 范围闭合 | 易顺手加新段/新表 | 红线钉读面零写+停条件 |
| 4 | 与票面零冲突 | bool 列既有语义未明 | 指令先读消费径再动 |
| 5 | 停止条件 | 数据面/枚举冲突两态 | 明列停或记取舍句式 |
