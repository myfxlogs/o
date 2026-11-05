# T-0356 O-Dev 施工提示词——volsell 族复役

你是[角色:施工]。领单授权=业主令外派。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围）：
   `/opt/o/docs/tasks/T-0356-volsell族复役-期权卖权限已开.md`
3. 病灶勿再诊；参考件先看：
   `cmd/research/dcd.go:159-205`（researchArms 装配面+dcd enabled 闸
   先例）、`internal/research/band/band.go`（白名单界桩——vol_sell
   钦定排除件勿动）、`cmd/core/volsell_open.go`（T-0215 executor）、
   `docs/tasks/T-0353*`（dcd 复役模板+条件书/四步启用序同款形）。

## 要点

- **根因先证**：eval 断流真因（连坐/凭据/目录面断）有锚再修——
  禁止先假写再对号；
- 复役照 dcd 模板：enabled 键（如需）钉 SG 正本非副本+fail_streak
  复役条件书落票面+启用步序落纸（旧码 boot 读键窗风险序写清）；
- **担保腿域分流是红线级**：期权卖=保证金占用，业主资产域腿走
  条件请求闸（T-0353 件④同款），提案落盒非 armed；
- vol_sell 阈档零动：0.06 纯阈保，白名单零改。

## 硬要求

- 新 worktree 先跑 `scripts/provision_worktree.sh`；
- 收口前 `scripts/check_all.sh` 全绿+戳对口提交 SHA；
- 领单/收口/移交 report.sh 里程碑自报全落；
- **不合并、不部署、不改票面**——超单即停回报；
- 撞限流先落 WIP commit+挂起自报再退（复工径：新 run 查 worktree
  半成品续作）。

## 移交口径

收口行 note 带：分支名+HEAD SHA+check_all 戳号+五验收锚逐条自证
（命令输出锚，禁口报；锚①根因证据行必填）。
