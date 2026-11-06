# T-0382 施工提示词——前端报价点与价格提醒

你是[角色:施工]。领单授权=DM 常径机派；收口回决策席常径裁。

按序读：
1. /opt/o/AGENTS.md + docs/collab.md + skills/施工席/SKILL.md
2. 票面唯一权威范围：
   /opt/o/docs/tasks/T-0382-前端报价点与价格提醒-飞书通知.md
3. 参照件：cmd/core/cockpit_face.go+web/src/react/pages/
   Cockpit.tsx（报价卡挂面）+ internal/platform/notify/
   venue_req.go（推送径同款）+ internal/portfolio/lifecycle/
   hunger.go（协程拍正径）+ web/src/react/components/flowpulse
   .tsx（T-0371 人话语义范式）

要点轴：md_ticks 已在采 BBO 全谱——数据面零新增，做报价卡
（符号自选）+fin_price_alert 表+前端设置点+o-core 提醒拍
（命中→飞书→fired 锁）。单向告警非条件单。

硬要求：新 worktree /opt/o-wt/t0382 + bootstrap 先跑→
check_all 绿戳对口收口 SHA→收口模板移交决策席；迁移号查
水位+1；前端测试照既有 DOM 测范式；不合并不部署不改票面。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 提醒拍挂哪个进程可选 | 票面钦定 o-core 协程照 hungerLoop | 禁新进程 |
| 2 | 提醒表名未定死 | 给 fin_price_alert 或按面谱 | 票面留口 |
| 3 | 防重响易漏 | fired 锁+桩测③钉 | 锚钦定 |
