# T-0389 施工提示词——驾驶舱版式解耦

你是[角色:施工]。领单授权=DM 常径机派；收口回决策席常径裁。

按序读：
1. /opt/o/AGENTS.md + docs/collab.md + skills/施工席/SKILL.md
2. 票面唯一权威范围：
   /opt/o/docs/tasks/T-0389-驾驶舱版式-瓦片间距与列联动解耦.md
3. 参照件：web/src/react/pages/Cockpit.tsx + web/src/react/
   styles/20-cards.css（版式容器/exch-sec 样式面）

要点轴：两症同根=列间不独立——改各列独立容器/对齐使列内
gap 恒定、展开只增本卡高。DOM 几何桩测断言「展开不扰他列」。

硬要求：新 worktree /opt/o-wt/t0389 + bootstrap 先跑→
check_all 绿戳对口→收口模板移交决策席；几何桩测+现有
domtest 回归绿；不合并不部署不改票面。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 容器结构未定 | 票面钦两径择（独立列/对齐修） | 工人择 |
| 2 | 联动根因未查 | 票面钦 DOM 几何断言 | 实证 |
