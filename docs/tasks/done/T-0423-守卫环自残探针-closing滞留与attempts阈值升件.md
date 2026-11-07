# T-0423 守卫环自残探针——closing 滞留/attempts 阈值升件（T-0419 范围 2b 遗留）

- 状态：完成（ops_event 完成行 2026-11-07 在案——机械轨票翻）
- 领取：无头施工席（devin o-worker-T-0423）@ 2026-11-07（支 t/0423-guard-close-loop @ /opt/o-wt/t0423）
- 优先级：P2——钱径监护面缺口（非新发 bug，循环已靠 gave_up 兜底但中途零告警）
- 收口移交归：决策席

## 背景（生产实证 2026-11-07）

fin_position #34 CORE-USDT-SWAP：delta_drift 幻觉触发平仓循环，
`fin_guard_close state='closing' attempts=5` 连绞烧手续费，
**中途零告警**——既有 `close_gave_up` 仅在 attempts+1≥max（当前
`guard.retry_max_attempts=5`）才火，循环前 4 轮纯静默。
T-0419 票面范围 2b 钦定此探针，收口件未实现且偏离说明未声明——
本票补立，原票根修件不受影响已并。

## 钦定语义

```
守卫自残枚举：state='closing' 且 (attempts≥N 或 now-last_try_at>Xh)
             →探针行+Alert 升件——平仓循环本身是事故，不等耗尽才说
取值钦定：   N、X 走 app_config dm 类阈键（guard.loop_alert_attempts /
             guard.loop_alert_hours 族，种子 N=3、X=2h——可自主调域）
```

## 范围

1. closing 滞留/超次探针：fin_guard_close `state='closing'` 且
   `attempts≥guard.loop_alert_attempts` 或
   `now()-COALESCE(last_try_at,建仓刻)>make_interval(hours=>guard.loop_alert_hours)`
   → Alert（kind=guard_close_loop 族）+ ops_event 行；同仓告警去重
   （已报仓不连拍重复刷——节流或幂等键自定）；
2. 阈键种子迁移：app_config 两键默认值写入（迁文件序号接续）；
3. 红桩：fixture closing attempts=4 或滞留>阈→升件行/Alert 断言
   （修前零行红、修后绿）；反向锚：attempts<阈且未滞留=零告警。
4. **熔断条款（业主令 2026-11-07 并入——「机器发现不对但不知
   为何，照开照烧」案补层）**：超阈触发时不只升件——同仓/同
   策略**挂起开仓权**（禁新开，平仓权保留=只许出不许进），
   解冻唯一径=裁定席判词或业主令；挂起态本身落 ops_event+
   前端可见面（禁静默挂起）。实现径：strategy 态迁/守卫
   黑名单键自定，票面不钦定。

## 验收

- 红桩：超次/滞留两形各断言升件迹，未达阈零告警；
- 熔断桩：超阈后同策略新开仓被拒（平仓单放行）+
  挂起态 ops_event 行断言；解冻后开仓权恢复；
- 阈键经 app_config 可调（直写 SQL 禁——生产调参走 tools/dmconfig）；
- 绿戳对口+全链。

## 结果回填

- 状态：待验（施工 session devin-o-worker-T-0423 @ 2026-11-07 收口）
- 实现落点：
  - 迁移 `fin_0138_guard_loop_suspend.sql`：fin_guard_close 加
    `loop_alerted_at`（探针行锚+节流戳）；新表 `fin_guard_suspend`
    （strategy_ref/position_id/reason/suspended_at/lifted_*
    三列，部分唯一索引=同策略至多一在挂行=幂等键）；阈键种子
    `guard.loop_alert_attempts="3"`、`guard.loop_alert_hours="2"`
    （guard.* dm 写域已册——dmconfig 直调径天然成立）；登记表
    migrate_table.go+domain.tsv 同步。
  - 探针 `internal/guard/loop.go`：`scanOnce` 内挂 `loopProbe`——
    `state='closing'` 且 `attempts≥N` 或 `now()-COALESCE(last_try_at,建仓刻)>Xh`
    →熔断（suspendOpen 幂等插+ops_event「开仓挂起」行）→节流窗内
    已报不连刷；出窗/未报→`loop_alerted_at` 落戳+`guard/loop_alert`
    计数+Alert(guard_close_loop)+ops_event 升件行
    （esc_class='守卫环'/esc_scope='fin_guard_close'，ticket=''——
    票面事件非票生命周期，不占票态轨）。阈键缺=fail-closed 跳过
    +missing_param 计数（哨戒主环不炸）。
  - 熔断拦点 `internal/risk/check.go`：Gate.Check open 径 intent
    完整性后查 `fin_guard_suspend` 在挂行→deny `guard_suspended`
    （挂起面读败=deny `suspend_read_err` fail-closed）；close 径
    不查=平仓权保留（只出不进）。闸=全径开仓单点（open.go×2/
    basis_open/simdrill 同闸）。
  - 解冻径 `guard.LiftSuspend`+`tools/guardlift`（dmconfig 同构薄壳）：
    -by 闭集 裁定席|业主 机械守→lifted_at/lifted_by/lift_reason
    落列+ops_event「开仓解冻」行。
  - 前端可见面 `cmd/core/flow_pulse.go`：position 工件 LEFT JOIN
    在挂行→Stage='开仓挂起'（无 proto 变更；挂起态非静默）。
  - humanize.go `guard_close_loop`→「平仓循环自残」人话注（事件级
    ——解冻归裁定席/业主令非业主三域动作）。
  - 数据流.md 守仓节同 commit 补「守卫环自残探针+熔断（T-0423）」行。
- 测试证据：
  - 红桩①（超次形）`TestLoopProbeAttemptsEscalate`：closing
    attempts=4>阈3→升件行/开仓挂起行/loop_alerted_at 戳/Alert 推
    送四迹断言；同窗二拍节流=1 不连刷；挂起策略 open deny
    guard_suspended、close 放行；非闭集主体解冻拒、裁定席解冻→
    开仓解冻行+open 恢复 allow——全绿。
  - 红桩②（滞留形）`TestLoopProbeStaleEscalate`：attempts=1 但
    last_try_at=3h>2h→升件/挂起/Alert 三迹——绿。
  - 反向锚 `TestLoopProbeUnderThresholdSilent`：attempts<阈且未
    滞留→零升件/零挂起/零推送，open 放行；阈键种子值机读断言
    （attempts=3/hours=2）——绿。
  - 全谱：check_all mode=full 绿戳（build/vet/import/gofmt/env/
    rejects/config-write-domain/migrations/bridge/tolerate/ops-chain
    +go test 全谱——O_TEST_PG_DSN o-postgres-test:5434）。
- 偏离说明：
  - 熔断实现径取「守卫黑名单键」（fin_guard_suspend 闸面拦）非
    策略态迁——票面两可自定；理由：Gate.Check=全径开仓单点
    （live/probe/dcd/basis/simdrill 同闸），态迁法需动六态 CHECK+
    逐面消费端知会新态，黑名单表零侵入且 lifted_* 自带审计列。
  - 「同仓/同策略」粒度=strategy_ref（同仓所属策略开仓权——position
    本身不可再开，禁新开的有效域即策略级；position_id 落列作审计锚）。
  - 探针 ops_event 升件行 ticket='' 而非 T-0423——fin_0111 登记臂
    对同 ticket 升件行做判例化，带票号会把本票裁决误登为「守卫环」
    判例；归属信息在 note（仓#/策略#）。
