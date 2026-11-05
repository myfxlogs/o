# T-0357 O-Dev 审计提示词——进水口零转换三径判词

你是[角色:施工]。领单授权=业主令外派（**本票=审计票只读+判词，
不写运行码**）。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围）：
   `/opt/o/docs/tasks/T-0357-进水口零转换审计-scout-xvenue-hypothesis三径判词.md`
3. 三径入口先看：`internal/research/scout/`（venue scout）、
   `internal/research/sigdist`+`hypothesis`（信号→假想官）、
   `internal/research/xvenuecarry` 或同名包（跨所 carry）——
   转换件（信号→提案）是审计焦点。

## 要点

- 逐径 trace+取证：信号产出在不在→阈判据达没达→提案写径通没通；
- 判词二分钉死：**正常空转**（数据到阈未达——市场不给件，
  记账销案）vs **断线**（数据断供/转换件死——静默病，
  给点位+修法建议）；
- 断线径给最小修法建议写回填，**不擅自修**——修票另立；
- 生产库只读查询为证据源（SG o-postgres 经既有只读径），
  查询输出锚落回填禁口报。

## 硬要求

- 新 worktree 先跑 `scripts/provision_worktree.sh`；
- 收口前 `scripts/check_all.sh` 全绿+戳对口提交 SHA
  （docs 面变更按门禁跑）；
- 领单/收口/移交 report.sh 里程碑自报全落；
- **不合并、不部署、不改票面、不改运行码**——超单即停回报；
- 撞限流先落 WIP commit+挂起自报再退（复工径：新 run 查 worktree
  半成品续作）。

## 移交口径

收口行 note 带：分支名+HEAD SHA+check_all 戳号+三径判词各一句
（正常空转/断线-点位）+证据锚清单。
