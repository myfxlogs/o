# T-0375 施工提示词——机制面静默缝四件

你是[角色:施工]。领单授权=业主手动外派（T-0375 票面领取行标
「业主手动外派」，审结归 COS 不归 DM——collab.md 写面判据窄口
豁免件）。

按序读：
1. /opt/o/AGENTS.md + docs/collab.md + skills/施工席/SKILL.md
2. 票面唯一权威范围：
   /opt/o/docs/tasks/T-0375-机制面静默缝四件-队列豁免识读与坟场退役口径.md

四件轴：
① scripts/meta/dm_queue.sh——`外派` 入已批分支排除集（外派件
   COS 审结轨接走，不占 DM 待裁队列；现每班复现）；
② scripts/bootstrap_worktree.sh 新件——新 worktree 一键再生
   internal/gen+web/src/gen（buf generate 正径，rsync 部署树为
   降级径明载）；AGENTS.md/施工席职卡各钉一行；
③ deploy_image.sh 完工回执——查 o-metabus 投件端点死活，修通
   或定性文档化（tolerate 定性刷新）；
④ scripts/meta/wt_gc.sh——audit-*/dec-t-* 系支退役口径（无 wt
   挂+末 commit>14d 即删），头注明载。

硬要求：新 worktree（/opt/o-wt/t0375）→check_all 绿戳对口
SHA→收口模板移交行写 COS（非决策席）；桩测自证（dm_queue
外派序列/wt_gc 两档龄闸）；不合并不部署不改票面；业务 SQL
语义零动（①只加排除词）；僵尸票定性不在本票。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | dm_queue 排除集改动=SQL 文本级 | 收窄为加词不改结构 | 票面红线注明 |
| 2 | bootstrap 涉生成件再生 | 正径 buf generate | 降级径写明 |
| 3 | 回执端点核查可能无修法 | 允许「核查结论落档」收口 | 验收锚③即此义 |
