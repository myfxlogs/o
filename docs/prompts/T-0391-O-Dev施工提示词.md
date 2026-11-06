# T-0391 施工提示词——自选报价条 NOX 移植

你是[角色:施工]。领单授权=DM 常径机派；收口回决策席常径裁。

按序读：
1. /opt/o/AGENTS.md + docs/collab.md + skills/施工席/SKILL.md
2. 票面唯一权威范围：
   /opt/o/docs/tasks/T-0391-自选报价条-NOX-QuoteStrip移植.md
3. 参照件：
   · 语义真源 /opt/nox/web/src/components/QuoteStrip.tsx
     + WatchlistModal.tsx（复刻语义非代码拷——NOX=React
     直订流帧,O=md_ticks 快照+ev_ticks 重取）
   · 本仓 web/src/react/components/QuoteBoard.tsx（嵌段位）
   · cmd/core/quotes_face.go（RPC 同位+fin_price_alert 审计式）
   · migrations/fin_0119_price_alert.sql（迁移范式+域钉）

要点轴：fin_0121 建表+Get/Set RPC（origin 闸+审计行）+报价条
段+浮层（搜索/点选/排序/全量替换保存）+提醒表单候选收口。
零成员/零行情如实缺；tile 形随现有 Card/Modal 件系。

硬要求：新 worktree /opt/o-wt/t0391 + bootstrap 先跑→
check_all 绿戳对口→收口模板移交决策席；不合并不部署不改票面。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 候选空=全采无行情 | 票面钦提醒表单回退全候选 | 照票 |
| 2 | NOX 帧流订阅径O无 | 语义换 md_ticks+ev_ticks 重取 | 票面钦 |
