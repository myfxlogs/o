# FEAT-01 收口：Earn 自动停泊开关（前端可切——治「停整机才能停扫额」缺口）

> 票面：外部-FEAT-01｜分支 `t/feat-earn-autopark`（worktree /opt/o-wt/feat-earn）
> 部署注：生效须 o-core 下次部署（可与 fin_0090 批搭——本票迁移=fin_0091）。

## diff 摘

- `migrations/fin_0091_earn_sweep_switch.sql`（新）+ `migrate_table.go` 册记 {91}：
  `INSERT app_config earn.sweep_enabled='true'`——默认开=部署不改行为。
- `internal/earn/worker.go`：`scan()` 在 pollRate 后、sweep 前置插
  `GetBool("earn.sweep_enabled")` 闸——false/读败/键缺→`rej.Inc("sweep_disabled")`
  +return（fail-closed 不动钱）；pollRate/RedeemToTrading 不闸。
- `proto/dashboardv1/core.proto`：`rpc SetEarnAutoPark{enabled}→{enabled,generation,ts}`
  单用途写径（禁泛化）；`GetCockpitResponse.earn_auto_park=8`。
- `cmd/core/earn_face.go`（新）：app_config upsert（generation 由 fin_0001 触发器
  seq 自涨——不手写防双跳）+`ops_event` 审计行（actor=cockpit、old→new、gen）。
- `cmd/core/cockpit_face.go`：`earn_auto_park`=`value::text='true'` 实读，败/缺=false
  （与闸同径 fail-closed，显实态非假绿）。
- `cmd/core/earn_face_test.go`（新）：锚③测试——写-读回-审计行-generation-复切。
- `web/src/react/pages/Cockpit.tsx`：KPI 条插「① 自动到活期」瓦——StateDot 态+
  确认弹层（window.confirm 钱径相邻确认）+pending/败态行内提示+切后 refetch。

## 锚逐项证

1. 闸行为：`scan()` 闸位与语义见 diff；单元实证——测试库迁移后 UPDATE
   earn.sweep_enabled=false→GetBool 返 false→scan 提前 return（pollRate 先行不受闸）。
2. RedeemToTrading 不闸：闸只包 sweepIdle+sweepResidual 两调用点——赎回径
   代码面零触碰（mover.go 未改）。
3. RPC：`TestSetEarnAutoPark` 绿——写→`value::text`='false' 回读+ops_event
   earn_autopark_toggle 行落+generation>0+切回 true（O_TEST_PG_DSN=5434 o_test）。
4. 前端：`npx tsc --noEmit` 绿、`npm run build` 绿（935ms）；瓦片=同源 Card/
   StateDot 件零新框架。
5. `scripts/check_all.sh` 全绿戳：见下（check-green-<HEAD>）。
6. 数据流正本：链路未变（config 写径既有 app_config+config_changed 链）——
  无需同 commit 更新，注记于此。

## 取舍记

- 确认层=window.confirm 原生弹层——钱径相邻件确认语义达成且零新框架；若决策
  席要模态件可后换（单点替换）。
- 审计行择 `ops_event`（seat=cockpit event=earn_autopark_toggle）——fin_risk_decision
  为风控裁决域非开关切换域，ops_event 同族（领单/相位自报同表）。
- 审计行写败=RPC 报错返回（开关已切——不一致窗告警>静默；败面窄）。
- fin_0091 跳号：fin_0090 他支（T-0267 ops_ticket_usage）已占——合并序依部署批；
  两表无交集零冲突。

## 读败语义裁定留档

`GetBool` err（键缺/型非）→闸视为 false 停扫（earn 包 fail-closed 同律）；
种子迁移保生产键在故恒 true——部署不改行为、显式切才变。
