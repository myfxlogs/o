# O-Dev 施工提示词 —— T-0348 venue 新品哨兵

> 派工依据：业主令（2026-11-04 23:4x）——「还是做外派」。此提示词即领单授权，
> 由 COS 代业主起草并自审过闸。票已 `已批` 在案，票面即范围，勿再诊断病灶。

## 你的身份与通用纪律

- 你是 `[角色:施工]`，一 session 一角。先读 `/opt/o/AGENTS.md`、
  `/opt/o/docs/collab.md`、`/opt/o/skills/施工席/SKILL.md`。
- 仓库 `/opt/o`；实现面**必须开 worktree**（master 仅容 docs/**，D-651 会拦）。
  worktree 起后先跑 `bash scripts/provision_worktree.sh`（补 gen/dist 依赖件）。
- 提交格式 `[角色:施工][T-0348] 摘要`；`bash scripts/check_all.sh` 绿才可收口移交。
- 里程碑 `~/bin/report.sh 领单|收口|移交 T-0348 "备注"`。
- 施工不自验自裁：收口后写 `移交` 行即停手等审计，不 merge 不 push 不部署。
- 撞限流体面停：写清进度 `report.sh 挂起` 退出，不死循环重试。

## 任务

票面：`/opt/o/docs/tasks/T-0348-venue新品哨兵-在册venue产品目录与公告diff探测.md`

已替你踩过的点（直接用，勿重摸）：

- **对形件**=`scripts/meta/gauges_tick.sh`：env 径读 FIN_DSN、
  `hb()` 写 `ops_headless`（本票心跳键 `META-VENUEWATCH`）、
  `ops_event` INSERT 径、取数败显形——整套抄形。
- **unit 对形件**=`~/.config/systemd/user/o-gauges-tick.{service,timer}`
  与 `o-wt-gc.{service,timer}`（今日新装可参考实件），仓内正本参照
  `deploy/o-econ-verdict.*`——**`deploy/` 正本 + `~/.config` 装机同 commit**，
  装完 `enable --now` 后 `list-timers` 截图锚收口（T-0331/0332 孤儿脚本
  教训：脚本不带电的部署锚已漏过一次，本票红线级要求）。
- **OKX 端点实探再写码**（协议面取证纪律）：先 curl 实测
  `/api/v5/support/announcements`（annType 参数族逐一试，把返回面摸清）
  和 `/api/v5/public/instruments`（instType 枚举全集：SPOT/MARGIN/SWAP/
  FUTURES/OPTION 等），以**真实响应**定探测字段——不许按文档猜。
  「简易期权」若两端点均不可见，如实记「API 不可见面」+核证过程；
  可见则首天然锚直接销。
- **幂等 seen-set**：存 `$ST` 文件（照 gauges_tick 的 /var/tmp 暂存径）
  或 PG 小表自裁——键=venue+公告 id/instId；同品重放零重复事件。
- 关键词集钉脚本头常量，新品类词面=期权/options/Earn 新品/新上产品族——
  **精确词面，勿宽正则**（公告面噪声大，宽匹配=事件刷屏）。

## 停止条件

- 端点需鉴权/凭据才可读 → 停，回报（票面红线=零凭据公开面）。
- 需动 `internal/`/`proto/` 等码面才装得下 → 停，回报（本票钦定纯脚本拍）。
- 票面范围判断有误 → 停，回报，不擅扩。

## 自审记录（提示词五查，起草方填）

| # | 查项 | 发现 | 处置 |
|---|---|---|---|
| 1 | 依据链 | 业主令原文+票已批在案 | 提示词首段即授权依据 |
| 2 | 指代歧义 | 「公告端点返回面」未知 | 钉实探先行+禁按文档猜 |
| 3 | 范围闭合 | 哨兵职责止于事件——易顺手写提案卡 | 红线明钉不碰 hypothesis 窄门 |
| 4 | 与票面零冲突 | timer 装机易被当部署外活 | 票面锚④明列装机实证为验收件 |
| 5 | 停止条件 | 鉴权墙/码面需求/票面误判三态 | 明列停+回报句式 |
