# T-0362 O-Dev 施工提示词——活水管道流动骨架

你是[角色:施工]。领单授权=COS 已批常票（业主裁 A 骨架档）。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围）：
   `/opt/o/docs/tasks/T-0362-活水管道流动骨架-纯前端.md`
3. 参考件先看：
   `web/src/react/components/flowpulse.tsx`（本件主体——七段
   瓦片+在途行）、`web/src/react/components/ui.tsx`（Card/
   状态件范式）、`web/src/react/pages/Review.tsx`（人话/
   排版范式参考）。

## 要点

- **信息层级是目的**：一瞥=「活/断/闲」——状态点四色做
  主信号，滞后人话做次信号；
- 判词行克制——从 state 序列机械派生（首个非中性非绿段
  =断点），禁造复杂语义；
- 脊线手画（flex 连接/SVG 线均可），**禁引 Pixi/图形库/
  新依赖**；
- FlowStage 字段之外零取——无新 RPC 无新查询。

## 硬要求

- 新 worktree 先跑 `scripts/provision_worktree.sh`；
- 收口前 `scripts/check_all.sh` 全绿+戳对口提交 SHA；
- 领单/收口/移交 report.sh 里程碑自报全落；
- **不合并、不部署、不改票面**——超单即停回报；
- 撞限流先落 WIP commit+挂起自报再退。

## 移交口径

收口行 note 带：分支名+HEAD SHA+check_all 戳号+五验收锚
逐条自证（锚①③截图或 DOM 输出锚）。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|------|------|------|
| 1 | 「判词」措辞未逐字钉 | 前端派生件 | 票面给了三类句式，工人按 state 序判 |
| 2 | 在途行（pipe-ticket）样式 | 范围外 | 票面未含——仅瓦片+判词行；在途行 T-0361 域 |
| 3 | 状态色值未钉 | 风格件 | 随 ui.tsx 现有色板（u-good/u-bad/灰阶） |
