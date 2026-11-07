# T-0428 平仓记忆补强——attempt 败因落表与 trigger 判词回写

- 状态：施工中（2026-11-07）
- 领取：外派施工（O-Dev 径，不占 DM 内联派工队列）——claude-t0428@2026-11-07（worktree /opt/o-wt/t0428，支 t/0428-close-memory）
- 优先级：P2——记忆完整性缺口（不挡钱径，属教训资产化）
- 收口移交归：决策席
- 关联：T-0420（postmortem 判词源——trigger_verdict 的写者）、
  T-0423（守卫环探针——同表域不并案）、pos34 实证
  （attempts=5 败因零落表+trigger_kind 记谎双缺口）

## 病灶（今晨取证实锚）

- `fin_guard_close` 无败因列——pos34 attempts=5 每次「为什么
  败」（Earn 收走余额→计划即败）只活在容器日志，容器
  重建即死：**教训不可复用，下案重演还得人肉考古**；
- `fin_position.trigger_kind='delta_drift'` 记谎——真相是
  单位错配幻觉，但原字段永久留档；0420 判词只新增
  lesson 案卷不回写原位：**复盘读档还会被假原因骗**。

## 范围

1. **`fin_guard_close` 增 `last_error`/`last_attempt_at` 明细化**：
   守卫每拍平仓尝试的败因落表（计划期败≠venue 期败分类——
   计划期=余额不足/参数歧义类，venue 期=venue 返回码）；
   写点=守卫重试环现有 attempt 递增同槽，零新拍；
2. **`fin_position` 增 `trigger_verdict`**：postmortem/裁定
   判词回写位——原 `trigger_kind` 不动（留原貌=审计链），
   `trigger_verdict` 记修正判（例 `delta_drift`→
   `unit_mismatch_hallucination`）；写点=0420 结案判定机
   同槽扩展（正常件=trigger 复核确认原判，异常件=修正名）；
3. **迁移**：fin 链下一空序号（票面写作时 0132 起——
   施工时先查 `ls migrations/fin_*.sql | tail` 取真号）；
   列全部可空零默认值——存量行零回填义务；
4. **读面**：前端/日报暂不钦定展示——本票只补写面，
   展示面另案。

## 验收

- 红桩：守卫模拟连败→`last_error` 逐次覆盖为最新败因
  （断言两次不同败因留下最新值）；
- 判词回写桩：postmortem 异常结案→trigger_verdict 落
  修正名，trigger_kind 原值不动（断言双存）；
- 迁移幂等+键域白名单过 check_migrations；
- 绿戳对口+全链。

## 结果回填

- 状态：待验（外派径收口——回 DM 裁决，不占派工额度）
- 实现（2026-11-07 claude-t0428，支 t/0428-close-memory）：
  - 迁移 `fin_0132_close_memory.sql`（真号实查 0132——origin/master 尾
    fin_0131；migrate_table.go 登记 {132}）：fin_guard_close 增
    `last_error`/`last_attempt_at`、fin_position 增 `trigger_verdict`，
    全可空零默认零回填（禁回填律遵票面）；幂等 ADD COLUMN IF NOT EXISTS。
  - 守卫写点（internal/guard/facts.go+guard.go）：`markClosing` attempt
    递增同槽落 `last_attempt_at`；败因落表带分类前缀——闸拒
    `plan:gate:<reason>`、sizer 现读败/对齐零 `plan:sizer:平仓量现读败/
    对齐零`（=Earn 收走余额类计划期败，state 不迁按节拍原样重试）、
    venue 执行败 `venue:<err>`（venue 期败返回码）；`last_error` 逐次
    覆盖为最新败因，平成不清（attempts>1 仓的记忆锚）。零新拍——全接
    现有 attempts 递增/markFailed 槽。
  - postmortem 判词回写（internal/ledger/postmortem.go settle 同槽）：
    正常件 `trigger_verdict='confirmed'`（复核确认原判）；异常件=机械
    修正码集（fast_close/settled_over/trigger_unknown/bare_bag 逗点连）；
    trigger_kind 原字段零触碰（审计链）；写败计数+log 不阻钱径（与监察
    案写同律）。裁定席语义修正（unit_mismatch_hallucination 类）为该列
    钦定后手写者——本票只接判定机槽（票面写点单列），裁定席覆写径另案。
  - 数据流.md 守仓节同 commit 补「平仓记忆（T-0428）」行（链路变更同拍）。
- 测试证据：
  - 红桩①（守卫连败覆盖）`TestCloseFailReasonOverwrites`：venue 败→
    `venue:venue 返回码 51009` 落表+last_attempt_at 非 NULL；sizer 败→
    覆盖为 `plan:sizer:`；平成后 last_error 保留——三拍全绿。
  - 红桩②（sizer 败径）`TestCloseSizerFailReasonLogged`：state 不迁
    （closing 待重试）+`plan:sizer:` 落表——绿。
  - 判词回写桩 `TestPostmortemTriggerVerdictWriteback`：异常件
    verdict=含 fast_close+settled_over 码集且 trigger_kind='funding_flip'
    原值不动（双存）；正常件 verdict='confirmed'——绿。
  - 实测（O_TEST_PG_DSN 5434 o-postgres-test）：guard 包败径四测全绿、
    ledger 全包 58.7s 绿、build/vet/gofmt 净。
- 偏离说明：
  - 票面红桩设想的「闸拒首败」不可复现——risk/check.go:22 契约 close 径
    kill_switch/上限不拦（拦平仓=存风险）；计划期败代表改取 sizer 败径
    （正是 pos34「Earn 收走余额→计划即败」的实体路径）。
  - 「异常件=修正名」在机械判定机的可落形=修正码集（语义修正名需裁定席
    认知——列注释+数据流行均记其覆写权，回写径另案不越单）。
- 机检：check_all full 档绿戳（迁移入变更面自动 full）——commit 后对口贴出。
