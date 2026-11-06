# T-0385 施工提示词——教训矿工拍

你是[角色:施工]。领单授权=DM 常径机派；收口回决策席常径裁。

按序读：
1. /opt/o/AGENTS.md + docs/collab.md + skills/施工席/SKILL.md +
   skills/裁定席/SKILL.md（pending→verdict 机件）
2. 票面唯一权威范围：
   /opt/o/docs/tasks/T-0385-教训矿工拍-事件账自动挖工程案pending.md
3. 参照件：internal/portfolio/hunger/hunger.go（慢拍正径+
   去重面）+ fin_econ_lesson 迁移与 pending 读件（grep 全仓
   fin_econ_lesson）+ internal/review/ 或裁定席裁流件

要点轴：ops_event 窗聚类→签名去重→达阈起草 eng 教训
pending→裁定席既有裁流入 KB。矿工只挖不写 KB（裁定席
判词径）。防灌：日额5+签名去重+无高信号沉默。

硬要求：新 worktree /opt/o-wt/t0385 + bootstrap 先跑→
check_all 绿戳对口→收口模板移交决策席；三形桩测+实况窗
回放断言（0380/0382 案应产出）+日额闸；不合并不部署。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 表位择（econ扩/新表） | 票面钦工人按面谱择 | 两径备 |
| 2 | 聚类阈取值 | 票面钦≥2或单件高危 | 默认值注 |
| 3 | 裁定席 eng 案扩域 | 票面钦裁流复用 | 不建新席 |
