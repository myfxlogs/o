# T-0390 施工提示词——ListPriceAlerts panic 根修

你是[角色:施工]。领单授权=DM 常径机派；收口回决策席常径裁。

按序读：
1. /opt/o/AGENTS.md + docs/collab.md + skills/施工席/SKILL.md
2. 票面唯一权威范围：
   /opt/o/docs/tasks/T-0390-ListPriceAlerts成功径nil-Err-panic根修.md
3. 参照件：cmd/core/quotes_face.go（病灶 112 行）

要点轴：Err() 为 nil 时调 .Error()=panic——先取后判；同仓
`.Err().Error()` 裸调审计；先红后绿（修前 panic 复现断言红→
修后同断言绿，T-0388 纪律）。

硬要求：新 worktree /opt/o-wt/t0390 + bootstrap 先跑→
check_all 绿戳对口→收口模板移交决策席；不合并不部署不改票面。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 同型裸调面未知 | 票面钦审计清单 | 工人扫 |
