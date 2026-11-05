# T-0371 O-Dev 施工提示词——段卡语义精修（闲字单显饿色+悬停全谱人话）

你是[角色:施工]。领单授权=业主手动外派（票面领取行标
「业主手动外派」，审结归 COS 不归 DM——collab.md 写面
判据窄口豁免件）。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围——段语义权威表/任务五条/
   验收五锚照核）：
   `/opt/o/docs/tasks/T-0371-段卡语义精修-闲字单显饿色与行数标单位.md`
3. 参考件先看：
   - `web/src/react/components/flowpulse.tsx`——pipe-card
     title/StateDot title/mid 渲点；
   - `web/src/react/components/pipeStatus.ts`——humanStatus
     翻译闸（T-0366 母件，扩展落点同文件）；
   - `cmd/core/flow_pulse.go`——段 key→表/节拍映射权威源；
   - T-0366 实现范式（人话律：机器词禁入前端任何层）。

## 要点

- 闲态大格只显「闲」+新饿色档（琥珀暖色系——灰=死/缺
  暖=饿/等色律）；拒因归悬停；
- 七段悬停照票面 C 表范式可读句（段义+节奏人话+行数
  单位）；事件驱动段（提案/执行）**禁显预期节拍**；
- 状态点悬停人话化+D 表黑话泄漏清零（armed/venue/
  scout/beat/水印裸词可见面归零，属性/类名除外）。

## 红线

- proto/查询/数据源/服务端态标串/灯色阈零动——纯渲染层；
- 七段等宽布局不破；
- 撞限流落 WIP+挂起自报再退；不合并不部署不改票面。

## 收口

新 worktree→分支 `t/0371-*`→前端测试+tsc+vite build+
check_all 绿戳对口 SHA→report.sh 三件→§8 模板照填
（**移交行写 COS 非决策席**）→推送分支即交付。
