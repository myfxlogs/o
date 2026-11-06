# T-0379 施工提示词——broker 接入标准径（MT5 族）

你是[角色:施工]。领单授权=DM 常径机派（业主令 2026-11-06 内循环）；收口回决策席常径裁。

按序读：
1. /opt/o/AGENTS.md + docs/collab.md + skills/施工席/SKILL.md
2. 票面唯一权威范围：/opt/o/docs/tasks/T-0379-broker接入标准径-MT5族凭据到paper全链.md
3. 能力账必读——**双层**：
   a) 能力面全谱（能做什么）：skills/mt-gateway（proto mt4 43RPC/
      mt5 57RPC×7 域 references 全档）+ skills/mt-accounts（账户
      生命周期判例——**验证通过才入库/状态机五态/SearchBroker
      RPC/IsInvestor 只读鉴别/session 就绪事件驱动零轮询**——
      接入拍设计直接抄这四件判例）；
   b) O 实测能力账（O 验过什么）：skills/mt5-gateway-admin +
      skills/mt5-account + skills/mt5-quotes-history +
      skills/mt5-orders-streams + skills/mt5-trading-mutating
4. 参照件：internal/portfolio/lifecycle/xvenuepair.go（激活复
   用径）+ internal/platform/notify/venue_req.go（推送面同款）
   + internal/research/scout/collect.go（registered 态消费面）
   + scripts/config-write-domain.tsv（bridge.* owner 域钉死）

要点轴：凭据落(readcred `<broker>.mt5.*`)→接入拍发现→四拍
(探测/映射草拟/map闸/激活)→fin_venue_candidate 状态机补
probed/mapped/paper 三态。MT5 族通用——写 broker-agnostic
码,EXR 3001157 当首案干跑探测+草案两段实证(不激活)。

硬要求：新 worktree /opt/o-wt/t0379 + bootstrap 先跑→
check_all 绿戳对口收口 SHA→收口模板移交决策席；迁移号按
master 现水位顺位（先查 schema_migrations 最大号+1）；
桩测自证四锚全绿；不合并不部署不改票面。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 凭证命名约定需钉 | 票面钦定 `<broker>.mt5.*` 键形 | 工人照钉 |
| 2 | map 域权属分歧 | owner 类键不降域,条件请求收口 | 降域仅附注另票 |
| 3 | 迁移撞号史 | 票面要求查水位+1 | T-0377 判例沿 |
| 4 | CEX 接入诱惑 | 范围外=不包 | 票面红线钉死 |
