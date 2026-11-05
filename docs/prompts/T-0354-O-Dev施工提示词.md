# T-0354 O-Dev 施工提示词——DM 班次制（事件 kick+drain 自退）

你是[角色:施工]。领单授权=业主令外派（「工程要严谨，同时也要省成本」双约束）。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围）：
   `/opt/o/docs/tasks/T-0354-DM班次制-事件kick与drain自退.md`
3. 病灶勿再诊——票面实查段已是终审证据，施工从修法起。

## 施工要点（票面五件）

- **件①kick**：权威源=ops_event 白名单事件行（移交/收口/收口待审/升件），
  report.sh 旁报只是 best-effort 不作事实源；listener 选最简件
  （bash 级或复用既有 daemon——禁新造重栈）；
- **件②drain**：会话内 scan→裁→rescan→空才退——班次语义核心件，
  会话期新到事件自然吸收；
- **件③单例闸**：双开防（kick 连发/兜底拍撞车），T-0343 事故同款防；
- **件④幂等**：裁前查已裁态，崩后重进不重复 merge；
- **件⑤兜底**：30min 拍保留原语义——kick 哑=现状非更差。

## 硬要求

- 新 worktree 先跑 `scripts/provision_worktree.sh`；
- 收口前 `scripts/check_all.sh` 全绿+戳对口提交 SHA；
- 领单/收口/移交 report.sh 里程碑自报全落；
- **不合并、不部署、不改票面**——超单即停回报；
- 省成本双约束：空队列零LLM kick 不触发+一批件冷启一次（验收锚④⑤）；
- 撞限流体面挂起退出；票面冲突/范围不够：停，写清报回，不擅自扩。

## 移交口径

收口行 note 带：分支名+HEAD SHA+check_all 戳号+六验收锚逐条自证（命令输出锚，禁口报）。
