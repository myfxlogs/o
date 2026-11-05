# T-0372 O-Dev 施工提示词——容错位静默组补报

你是[角色:施工]。领单授权=业主手动外派（票面领取行标
「业主手动外派」，DM 不会自动领——你领）。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围）：
   `/opt/o/docs/tasks/T-0372-容错位静默组补报-usage_esc复用推谱.md`
3. 机制母件先看（照抄不发明）：
   - `scripts/meta/dm_tick.sh` usage_esc 节——T-0369 首用
     自报升件范式（带类升件+note 前缀同窗去重）；
   - `scripts/meta/escalate.sh`——带类升件径（esc_class/
     esc_scope 列）；
   - `docs/容错位清单.md` §2——七组静默位逐位病灶；
   - `docs/工程纪律.md` 容错必报条+`scripts/meta/
     check_tolerate_report.sh`——锚/registry 规则。

## 要点

- 七组静默位逐位接自报径——**复用 usage_esc 同式**
  （ops_event 升件/写败行+note 前缀 24h 同窗去重）；
  dm_tick 三位（73 队列败=空集头号/146 派单查询/
  168 看门狗本体）同文件同批落；
- usage_esc 需提共享件（dm_tick/dev_run 两用）走
  scripts/meta 面，**禁新通知栈**；
- 每修一位同步清单定性档（❌静默→✅知情）+
  `--emit-registry` 机读块刷新；
- 容错语义零动——败了仍容错，只加自报；
  半知情/末环位（escalate.sh 本体/report.sh 族）不修。

## 验收锚（票面五条照核）

- 七位全接或逐位写明「留静默」裁定理由；
- dm_queue 败注入实证（队列查询败→升件行落账非空集
  滑行）；
- dev_run usage 写径败注入实证（T-0369 锚③同形）；
- 清单定性档+registry 重生成零 diff 漂移；
- check_all 全链绿戳对口提交 SHA（含 tolerate-report
  lint）。

## 红线（票面节照录）

- 不动队列推导/相位语义/事件词表；
- 自报全走既有 ops_event/escalate.sh 白名单径；
- 凭据域/钱径/部署面零触；
- 撞限流落 WIP+挂起自报再退，不合并不部署不改票面。

## 收口（本件=手动外派，审结归 COS 不归 DM）

新 worktree provision→分支 `t/0372-*`→check_all 全绿戳
对口 SHA→report.sh 领单/收口/移交全落→结束语模板
§8 照填（验收锚逐条 PASS 证据+涉及面+范围外确认，
**移交行写「移交: COS（业主手动外派审结豁免
collab.md 写面判据窄口）」非决策席**）→推送分支即
交付毕，审结由 COS 执行。
