# T-0355 O-Dev 施工提示词——敞口闸产品仓读面根修

你是[角色:施工]。领单授权=业主令外派。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围）：
   `/opt/o/docs/tasks/T-0355-敞口闸读面根修-产品仓计入exposure_cap.md`
3. 病灶勿再诊；`internal/ledger/ledgered.go` 的 ReconSets/dcd_baseline
   聚合径与 `fin_position_view`（fin_0112）定义先看再动手。

## 要点

- exposure_cap 敞口计口径对齐统一持仓读面——施工按现构裁最贴点位
  （视图同径或凭据 Σamt 加项皆可，但语义必须=产品仓入闸）；
- **tick 缺价=fail-closed 钉死**——闸面与显示面口径相反是钦定语义
  （估不出敞口宁可拒放），勿取 cockpit 虚计零的口径；
- exec 仓原径零动。

## 硬要求

- 新 worktree 先跑 `scripts/provision_worktree.sh`；
- 收口前 `scripts/check_all.sh` 全绿+戳对口提交 SHA；
- 领单/收口/移交 report.sh 里程碑自报全落；
- **不合并、不部署、不改票面**——超单即停回报；
- 撞限流先落 WIP commit+挂起自报再退（复工径：新 run 查 worktree
  半成品续作）。

## 移交口径

收口行 note 带：分支名+HEAD SHA+check_all 戳号+四验收锚逐条自证（命令输出锚，禁口报）。
