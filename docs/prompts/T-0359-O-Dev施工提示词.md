# T-0359 O-Dev 施工提示词——IBKR 凭据读面 adapter（paper 先行）

你是[角色:施工]。领单授权=COS 已批常票（venue 引入方向业主已裁）。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围）：
   `/opt/o/docs/tasks/T-0359-ibkr读面adapter-凭据落地开接.md`
3. 参考件先看：
   `docs/venue/ibkr-凭据路书.md`（gateway 装设/首登/daily 重登
   全步序+出处）、`docs/venue/ibkr-api-对照.md`（CP Gateway 钦定
   理由）、`internal/data/okxpriv/collector.go`（私有读面形态
   先例——md_account_events 事实行+心跳活性行+键域归一）、
   `cmd/research/readcred.go`（app_secrets 解密正径）、
   `tools/secput/main.go`（秘件写入径——凭据已在库勿动）、
   T-0352 读败升件闸/T-0353 条件盒先例（复用禁新造）。

## 要点

- **凭据面**：`app_secrets` 三键 `ibkr.paper_username/
  paper_account/paper_password` 已封存——只许 `cryptoutil.Open`
  径解，明文零落面（argv/env/log/文件全禁）；
- **gateway 部署**=OCI 机 systemd user unit；首登 localhost:5000
  业主网页+IB Key——那是部署后验步，施工只管装设件与 runbook
  步序，不替业主登；
- **会话到期 fail-loud**（okxpriv 同款纪律）——到期/读败升
  T-0352 闸+条件盒落「待业主重登」，禁静默重试吞掉；
- **重登税评估是硬产出**：CP daily 2FA vs IBGW auto-restart vs
  其他——选项+代价落 runbook 供裁，别默认拍；
- 候选升态 `candidate→credentialed` 迁移随件落（编号续 md_
  序列）。

## 硬要求

- 新 worktree 先跑 `scripts/provision_worktree.sh`；
- 收口前 `scripts/check_all.sh` 全绿+戳对口提交 SHA；
- 领单/收口/移交 report.sh 里程碑自报全落；
- **不合并、不部署、不改票面**——超单即停回报；
- 撞限流先落 WIP commit+挂起自报再退。

## 移交口径

收口行 note 带：分支名+HEAD SHA+check_all 戳号+六验收锚逐条
自证（命令输出锚，禁口报；锚④凭据零明文 grep 输出必填）。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|------|------|------|
| 1 | 首登 localhost:5000 需业主浏览器+2FA | 部署后验非施工域 | 票面锚⑤ runbook 步序落纸，提示词明示「不替业主登」 |
| 2 | 凭据已在 app_secrets | 勿再写入 | 提示词列 secput 为参考非任务 |
| 3 | Futures-US 冷静期中 | 期货面不可达≠阻塞 | 票面明示本票不依赖期货权限 |
| 4 | 重登日税无钦定解 | 歧义件禁自裁 | 任务③列「选项+代价落 runbook 供裁」 |
| 5 | md_ 迁移号续编 | 执行细节 | 提示词「编号续 md_ 序列」一句带过 |
