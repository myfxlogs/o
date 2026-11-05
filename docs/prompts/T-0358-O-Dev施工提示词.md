# T-0358 O-Dev 施工提示词——信息面与钱径分域标记

你是[角色:施工]。领单授权=业主令外派。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围）：
   `/opt/o/docs/tasks/T-0358-信息面与钱径分域标记-日历econ推送归位.md`
3. 消费面先查：`docs/数据流.md` 推送面章节+cockpit 前端推送面
   展示处（有则改无则明示）。

## 要点

- 分域标注：notify.* 族「钱径事件 vs 信息面」两列归位——
  日历/econ_digest/regime 归信息面列；EventPush 三事件+venue_req
  归钱径列；
- 配置键口径钉正：三信息面键标「信息面非钱径」防下游审计再混计；
- **只动文档/标注面**——推送行为/开关/功能码零动。

## 硬要求

- 新 worktree 先跑 `scripts/provision_worktree.sh`；
- 收口前 `scripts/check_all.sh` 全绿+戳对口提交 SHA；
- 领单/收口/移交 report.sh 里程碑自报全落；
- **不合并、不部署、不改票面**——超单即停回报；
- 撞限流先落 WIP commit+挂起自报再退（复工径：新 run 查 worktree
  半成品续作）。

## 移交口径

收口行 note 带：分支名+HEAD SHA+check_all 戳号+四验收锚逐条自证
（diff 锚，禁口报）。
