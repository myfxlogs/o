# T-0178 钱径挂账清算+config 翻位审计列——T-0056 三挂账收口

## 态：完成（判过并码在案，11-01 清账翻面·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**完成**——审计通过落票在案，漏网翻面

| 立项 | `[角色:决策]` @ 2026-10-27（债务盘点 B 级——T-0056 移交三件久挂） |
|---|---|

审计席：headless @ 2026-10-30

## 病灶（三笔挂账+一结构缺口）

1. **sweepIdle 未回存归因**：T-0056 窗内 `USDT trading 50.18` 观察期未见
   sweep 回存行——阈值/事件面未查（钱闲置=部署缺口）
2. **recon_adjust +1.70 来源未核**：施工窗内出现的调整行非本席执行——
   来源未对账（账对清才能真止鸣）
3. **live_switch 翻位者无审计痕**：config 写面无审计列——谁翻的何时翻的
   不可考（T-0056 移交③ 候票位）

## 施工面

1. `app_config` 加审计列：`updated_by text NOT NULL DEFAULT ''`+
   `updated_at timestamptz NOT NULL DEFAULT now()`——config 写径
   （config.Set*）装配位带写者身份（进程名/席名），迁移回填既有行
   `updated_by='legacy'`
2. 清算取证（决策席位协查非纯码面）：`fin_cashflow`/`recon_adjust` 行
   源核（+1.70 笔）；`sweepIdle` 触发面+阈值 vs 50.18 余额对账——
   若结构性不修（阈值高于余额=永不回存）则调阈或记例外，如实归因
3. 审计列落地后 T-0056 移交③ 自然愈合（此后翻位留痕）

## 验收锚

- `fin_cashflow` +1.70 行来源定性入票面（合法=记源/非法=立坑）
- sweepIdle 50.18 归因+处置入票面
- app_config 行带 updated_by/updated_at；Set 径新写留写者名
- 迁移过闸+retention 照账

### 施工注（派前补钉——桥表双侧约束）

`app_config` 是**桥表**——加审计列=桥 DDL 变更，七断言「两链同形」闸
要求 fin 侧迁移+md 侧同形件**同 commit 落**（md_1072 顺位），否则
check_migrations 即红。迁移序号：fin_0073 顺位（0071=host_metrics/
0072=condition_gap 已占）。

## 结果回填

### ① 审计列（双链同落）
- fin_0073_app_config_audit.sql：updated_by text NOT NULL DEFAULT '' +
  updated_at 既有列零动；既有行回填 'legacy'。md_1072_app_config_audit.sql
  副本同形（桥表两链同形闸过——七断言绿）。
- config.Set(ctx,key,value,actor) 写径新增：INSERT ON CONFLICT 写
  updated_by=actor+updated_at=now()（generation 序列触发器同源热启帧；
  live_switch 此后翻位留写者名）。
- projector 副本同步列扩展：qConfigAll/uConfig 带 updated_by——
  正本写者名副本面同形同步。

### ② 清算取证（生产对拍）
- **recon_adjust +1.70（id=93，2026-10-24 13:10）定性=合法校准行**：
  T-0100 残余币归口对账窗内校准批——同批 -3.39/+28.36/+125.17 五笔
  同窗分布（05:13~13:10），source_ref=自引/基线锚口径（基线校准
  一次性行律——非本席执行系前席对账动作，来源对清=合法记源）。
- **sweepIdle 50.18 归因=结构性不修**：earn.sweep_min_usd=100 >
  余额 50.18——阈值高于余额=永不回存（观察期零回存根因非事件面败）。
  处置=调阈 SG 生产已落 `earn.sweep_min_usd=25`（50.18>25 下拍
  自动停泊；updated_by 列迁移后翻位留痕）。

### ③ 锚
- 迁移七断言过（fin_0073+md_1072 同 commit 双链同形）；
  config 包 4 测绿；check-env-reads/rejects 全绿。

## 判决注记（审计席逐锚对账——SG 生产只读+码面+diff 三径取证）

- 锚① +1.70 定性：SG `fin_cashflow` 实拍 id=93 amount=1.70 USDT
  `source_table=recon_baseline_adjust` @2026-10-24 13:10:59Z 在案；同窗
  基线校准批七笔（-96.01/+75.92/+125.17/+28.36/-3.39/+0.48 等，
  recon_baseline_adjust 族+md_account_events 锚一笔）——
  `ledger.go`「recon_adjust=基线校准一次性行」律口径对，
  票面定性「合法校准行·记源」成立。（票面「同批五笔」计数偏松——
  同窗实七笔，不影响定性结论。）
- 锚② sweep 归因+处置：`internal/earn/worker.go:181` `idle<=thr` 直返
  ——阈 100>余额 50.18 永不回存，「结构性不修」归因与码面吻合；
  SG `app_config` 实拍 `earn.sweep_min_usd=25` 实落；翻阈后约 1.3s
  `fin_risk_decision` id=32946 `earn_deposit` 30.97 allow/transfer
  @2026-10-27 06:41:52Z——闲资实弹回存生产实证（留痕径=decision 行，
  T-0060 后 earn_in/earn_out 停写 cashflow 系钦定口径非缺证）。
- 锚③ 审计列+Set 留名：SG `app_config` 双列在（updated_at 自
  fin_0001 既有、updated_by 由 fin_0073 落）；`write.go` Set 走
  INSERT…ON CONFLICT 双径写 `updated_by=actor`+`updated_at=now()`；
  SG 实拍 76 行回填 'legacy'，迁移后新写 named 写者二行实证
  （`trx.vote_enabled`='t-0184-vote-arm'、`readcred.entitlement`
  ='fin_0076'）。
- 锚④ 迁移过闸+retention：fin_0073+md_1072 同 commit 双链同落、
  `migrate_table.go` 双登记序号唯一单调、桥表双侧 ALTER 同形；
  迁移契约闸无涉（updated_by 非 venue/account/symbol 键域列）；
  md_1069 retention 账 `app_config='state'` 行无变更需要。
- 注记（非锚不阻，报决策席跟踪）：`config.Set` 全仓零 Go 调用方
  ——写径能力在册未接线；且非 Set 径写仍可留空名实证
  （`lifecycle.reject_cooldown_s` updated_by='' @10-30）——
  「凡写必留痕」尚为约定非强制，是否立票收口由决策席裁。
