# T-0392 施工提示词——activeSet 现货腿派生补臂

你是[角色:施工]。领单授权=DM 常径机派；收口回决策席常径裁。

按序读：
1. /opt/o/AGENTS.md + docs/collab.md + skills/施工席/SKILL.md
2. 票面唯一权威范围：
   /opt/o/docs/tasks/T-0392-activeSet现货腿派生缺-funding-hedge失明根修.md
3. 参照件：internal/data/okxws/subscribe.go（activeSet UNION
   75-83 行+归一链）;internal/portfolio/lifecycle/opener.go
   （fundingHedgeFamily 集——模板属判据同源,禁自造清单）

要点轴：UNION 第四臂——fundingHedgeFamily 族 inst_id 剥 -SWAP
尾派生现货对;归一语义照 dcd/basis 臂;虚构键照既有 valid/
isUSDTInst 闸滤。

硬要求：新 worktree /opt/o-wt/t0392 + bootstrap 先跑→
check_all 绿戳对口→收口模板移交决策席；不合并不部署不改票面。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 族集出处双源险 | 票面钦 fundingHedgeFamily 同源引用 | 禁自造 |
