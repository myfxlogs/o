# T-0393 施工提示词——探针发现→自动起草票管

你是[角色:施工]。领单授权=DM 常径机派；收口回决策席常径裁。

按序读：
1. /opt/o/AGENTS.md + docs/collab.md + skills/施工席/SKILL.md
2. 票面唯一权威范围：
   /opt/o/docs/tasks/T-0393-探针发现自动起草修复票管.md
3. 参照件：internal/portfolio/hunger/（探针同槽位+ops_event
   写径）、scripts/next_ticket.sh（草案取号径——候选票编号
   是否走同一取号器票面未钦,工人自审裁定并在回填注）、
   docs/tasks/T-0384 票（族沉默探针签名范式）。

要点轴：升件行→签名聚类→候选草案文件+起草 ops_event 行;
签名级 dedup;fail-closed;只产草案不派发(权序红线)。

硬要求：新 worktree /opt/o-wt/t0393 + bootstrap 先跑→
check_all 绿戳对口→收口模板移交决策席；不合并不部署不改票面。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 候选票取号径未钦 | 自审裁定=同一取号器+候选态标 | 回填注 |
| 2 | 草案撞 T-0219 闸 | 候选态入 docs/tasks 照闸过号 | 同律 |
