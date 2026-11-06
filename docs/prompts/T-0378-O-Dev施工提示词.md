# T-0378 施工提示词——hungertick 接线

你是[角色:施工]。领单授权=业主手动外派（票面领取行标「业主
手动外派」；收口回决策席常径裁——86efad3 新规）。

按序读：
1. /opt/o/AGENTS.md + docs/collab.md + skills/施工席/SKILL.md
2. 票面唯一权威范围：
   /opt/o/docs/tasks/T-0378-hungertick接线-饥饿感知器挂lifecycle周期拍.md
3. 参照件：internal/portfolio/hunger/hunger.go（New/Detect/
   Respond 三拍签名）+ internal/portfolio/lifecycle/run.go
   （现有拍结构）

要点轴：hunger 包是死件零 caller——挂进 lifecycle 周期拍
（Detect 先 Respond 后,每拍至多响应一次,败隔离+容错必报
自报行）。接线非重写,hunger 本体零动。

硬要求：新 worktree（/opt/o-wt/t0378,bootstrap 先跑——
scripts/bootstrap_worktree.sh 已落地）→check_all 绿戳对口
SHA→收口模板移交行写决策席；桩测自证（触发构造+败径隔离）；
不合并不部署不改票面；镜像面件合批窗。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 接线位拍节奏不明 | 票面钦定照现有拍结构 | 禁野 goroutine |
| 2 | 败径易静默 | 容错必报律钉 ops_event 行 | 锚③桩测 |
