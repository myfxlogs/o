# T-0428 平仓记忆补强——attempt 败因落表与 trigger 判词回写

- 状态：已批（业主令 2026-11-07——「增强 memory，出外派单
  不给内联压力」）
- 领取：外派施工（O-Dev 径，不占 DM 内联派工队列）
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

- 状态：待派（外派径——DM 不占派工额度）
