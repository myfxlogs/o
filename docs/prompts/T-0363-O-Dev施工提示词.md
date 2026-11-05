# T-0363 O-Dev 施工提示词——敞口帽分母口径三病灶（钱安全卡对账）

你是[角色:施工]。领单授权=COS 已批常票（业主实数对拍例行
代理件，P2 显示真值——「钱安全吗」卡不许报假账）。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围）：
   `/opt/o/docs/tasks/T-0363-敞口帽分母口径三病灶-钱安全卡对账.md`
3. 主战场：`cmd/core/cockpit_face.go:76-172`（钱面/平台分列/
   敞口帽三段）；对拍件：`cmd/core/cockpit_face_test.go`、
   `cmd/core/service.go:132`（cashflow venue 读径范式）。

## 要点

- **病灶①双计**：`Capital`=全局 cashflow deposit+withdraw 净
  （含 mt5 deposit ~3,247 基线），`mrow.Deposited += a.Equity`
  再加一遍——修法=cashflow 求和滤 `venue` 出 mt5（列已在案，
  `coalesce(venue,'')` 读径范式见 service.go:132），MT5 侧只认
  mt5Live equity；
- **病灶②equity 虚胖先证后修（硬序）**：
  `fin_cmc_account_summary.total_account_value=10,517` vs 业主
  实证 $3,248——先查 mt5Live 取数径与 summary 写源
  （account_id 对错户/币种单位/快照龄三疑向），**根因判词落
  票面+收口 note**；未判明前禁改取数径；
- **病灶③recon_adjust 入分母**：kind 集扩
  `('deposit','withdraw','recon_adjust')`——对账真钱属本金语义；
- **三态照真值出**：修后卡以业主实数对拍（okx 10,126.23 +
  mt5 ~3,248 → Capital≈13,374±零头）；
- 禁造新口径——敞口/committed/parked 三段既有语义零动。

## 硬要求

- 新 worktree 先跑 `scripts/provision_worktree.sh`；
- 收口前 `scripts/check_all.sh` 全绿+戳对口提交 SHA；
- 领单/收口/移交 report.sh 里程碑自报全落；
- **不合并、不部署、不改票面**——超单即停回报；
- **闸面零动**：`risk.*` 参数/`fin_position` 写径/闸逻辑全禁
  （闸帽调值是业主另裁件）；只动读面求和/取数径，venue 零
  mutating；
- 撞限流先落 WIP commit+挂起自报再退。

## 移交口径

收口行 note 带：分支名+HEAD SHA+check_all 戳号+五验收锚逐条
自证——①Capital≈13,374 对拍实测值；②equity 根因判词（判词
件非口报：附取数径/写源查询实证+修后 live≈3,248）；
③recon_adjust 入分母实证（合成行或现值对拍）；④双计消除
（mt5 不二次入）实证；⑤绿戳对口 SHA。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|------|------|------|
