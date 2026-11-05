# T-0361 O-Dev 施工提示词——「找平台」卡降级

你是[角色:施工]。领单授权=COS 已批常票（业主亲指显示面件）。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围）：
   `/opt/o/docs/tasks/T-0361-找平台卡降级-进水口件非业主面.md`
3. 参考件先看：
   `web/src/react/components/venueboard.tsx`（本件主体）、
   `web/src/react/pages/Review.tsx`（ReportsCard/FlowsCard 的
   10 行 pager+`<details>` 收合范式——同款抄）。

## 要点

- **降级是正名不是美化**：卡=机器进水口件默认收合露计数
  摘要行；「建议注册」是唯一业主动作态→计数>0 时 summary
  显眼；
- 排序序钉死：`建议注册>候选>度量中>其余`，同态净年化降序；
- 分页/pager 抄同页范式不新造；
- 纯渲染层——scoutv1 Board 契约字段零动，禁顺手改查询。

## 硬要求

- 新 worktree 先跑 `scripts/provision_worktree.sh`；
- 收口前 `scripts/check_all.sh` 全绿+戳对口提交 SHA；
- 领单/收口/移交 report.sh 里程碑自报全落；
- **不合并、不部署、不改票面**——超单即停回报；
- 撞限流先落 WIP commit+挂起自报再退。

## 移交口径

收口行 note 带：分支名+HEAD SHA+check_all 戳号+五验收锚
逐条自证（锚①②前端态实证——截图或 DOM 查询输出均可）。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|------|------|------|
| 1 | 「显眼色」未指名色值 | 风格件走现有 ui 类 | 票面只钉「显眼」，类名/色值随 ui.tsx 现件 |
| 2 | 计数摘要口径（按 venue 数还是行数） | 票面未钉 | 按 venue 计数（业主关心「几个平台」非族行数）——提示词补此句 |
| 3 | 收合默认态会不会藏住建议注册 | 反例 | summary 计数>0 显眼=已防（任务①） |
