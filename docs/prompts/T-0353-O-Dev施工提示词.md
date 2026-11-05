# T-0353 O-Dev 施工提示词——中间档产线开闸

你是[角色:施工]。领单授权=业主裁 B 批后外派（业主批行在 ops_event 案）。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围）：
   `/opt/o/docs/tasks/T-0353-中间档产线开闸-阈值族级化与dcd臂复役.md`
3. 裁决背景（语义锚非范围源）：`docs/cos/packs/2026-11-05-middle-band-production-line.md`
4. 病灶勿再诊——票面实查段已是终审证据，施工从修法起。

## 施工要点（票面四件）

- **件①阈族级化**：近确定 carry 白名单族 4% 档、其余保 6%——先摸清
  现阈存址（config 键/模板列）再选最贴点位，族映射显式可枚举；
- **件②dcd 臂复役**：`research.dcd.enabled=true` 写 **SG fin 正本**
  app_config——PG-md 副本收割坑在案（DM 10-31 驳回行可查），禁写副本；
- **件③复役条件书**：票面钦三触发（臂连败>N、端点族死、业主令）——
  机器可判的写成机器判据+升件行，人判的留票面；
- **件④BTC 业主资产域**：base=BTC 的 dcd_call 提案止 `fin_condition_gap`
  （条件请求径现成——fin_0108 已建册），不进自动 armed；USDT 腿 dcd_put 常径。

## 硬要求

- 新 worktree 先跑 `scripts/provision_worktree.sh`；
- 收口前 `scripts/check_all.sh` 全绿+戳对口提交 SHA；
- 领单/收口/移交 report.sh 里程碑自报全落；
- **不合并、不部署、不改票面**——超单即停回报；
- venue 全零 mutating；撞限流体面挂起退出；
- 票面与设计冲突/范围不够：停，写清报回，不擅自扩。

## 移交口径

收口行 note 带：分支名+HEAD SHA+check_all 戳号+六验收锚逐条自证（命令输出锚，禁口报）。
