# T-0177 fin_condition_gap 请求盒建表——D-0014 机制承载落地

## 态：完成（判过并码在案，11-01 清账翻面·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**完成**——审计通过落票在案，漏网翻面

| 立项 | `[角色:决策]` @ 2026-10-27（债务盘点 A 级——D-0014 钉了机制、表从未迁；
业主接口面=唯一钦定交互径，现走会话） |
|---|---|

## 病灶

总纲「业主接口收窄=条件请求响应」的机制在 D-0014 钉为钦定位，但承载表
`fin_condition_gap` **从未建**——凭据缺/权限缺/条件缺时各臂只能打日志
（`[cmc] 凭据键缺——采集臂让位（业主条件请求件——对冲腿显读件缺）`
这类行就是孤儿请求），业主响应面无归口，系统也无法追踪「缺什么缺多久」。

## 施工面

1. 迁移件 `fin_0072`（0071 已被 T-0173 host_metrics 占）（fin 域——裁决/请托属钱径面非行情）：
   `fin_condition_gap(id bigserial PK, kind text NOT NULL CHECK
   (kind IN ('capital','credential','permission','constraint','holding',
   'executor')), venue text, surface text, payload jsonb NOT NULL
   DEFAULT '{}', state text NOT NULL DEFAULT 'pending' CHECK (state IN
   ('pending','fulfilled','rejected','dismissed')), requested_at
   timestamptz NOT NULL DEFAULT now(), resolved_at timestamptz,
   resolve_note text NOT NULL DEFAULT '', UNIQUE(venue,surface,kind)
   WHERE state='pending' 局部唯一)`——同面同类未决不重复开
2. 平台件 `internal/platform/condreq`（或归口既有窄包）：`Request(
   pool,kind,venue,surface,payload)` upsert 语义（pending 撞=刷新
   requested_at+payload 非重行）+`Resolve` 径
3. 接两处现存断点：`cmcArm` 凭据缺让位（trx.go）改发 `credential` 行；
   `fin_venue_untradeable`/venue 凭据检查同族位（票面允许最小两接点，
   全面接流后续票）
4. 读面：o-core ops face 已有多表读口径——挂只读查询面（前端面随 T-0173
   席次，不在本票）

## 验收锚

- 迁移过七断言闸；`fin_data_retention` 照账入册（不清账清道夫拒动）
- 桩锚：缺凭据态起 cmc 臂→`fin_condition_gap` 出 `credential/cmc` 行；
  补凭据后行 `fulfilled`（人工 UPDATE 口径可，自动关闭后续票）
- `check-env-reads`/`check_rejects_keys` 绿

## 审计判决

审计席：headless @ 2026-10-30

逐锚对账（收口=merge a2ae478 全量 diff 亲核——trx.go 接点/condreq
新包 29+41 行/fin_0072 迁移/migrate_table{72}+domain.tsv 登记）：

1. 锚① 迁移过七断言闸+fin_data_retention 照账入册——达，宣称件三证齐：
   本席 worktree 实跑 `scripts/check_migrations.sh` →
   「OK: check_migrations（七断言过——链内唯一/序号实存/零孤儿/md_*
   键域/两链单调/八桥 DDL 同形/域白名单）」；宣称件实存
   migrations/fin_0072_condition_gap.sql（20 行）列谱逐字对账：
   kind 六值 CHECK/state 四值 CHECK/payload DEFAULT '{}'/
   UNIQUE(venue,surface,kind) WHERE state='pending' 索引
   fin_condition_gap_pending_uq/retention INSERT permanent
   ON CONFLICT 幂等；登记表 migrate_table.go {72,…}+domain.tsv
   `fin_condition_gap fin` 双册在案；SG（o-postgres:5433 只读实查）
   schema_migrations version=72 在账、\d 实表 DDL 与迁移件逐项
   一致、fin_data_retention 行 class='permanent' 在册（清道夫拒动
   面封）。
2. 锚② 桩锚——达，且产面实证强于桩面：
   本席实跑 `go test ./internal/platform/condreq -run
   TestRequestUpsertResolve`（O_TEST_PG_DSN=localhost:5434 o_test
   独立实例）→ 1 passed——同键两行落库=单行（upsert 刷新非重行）
   +Resolve→fulfilled 两腿同桩验；接点实读 trx.go:160——cmcArm
   凭据三门（email/pass/acct==0）让位分支内嵌
   condreq.Request(credential,cmc,cmc.email+cmc.password+
   cmc.account_id) 与孤儿日志同框；SG 产面实查 fin_condition_gap
   id=1 行 credential/cmc/pending，surface 与载荷逐字同接点
   （{"missing":"凭据/账户号三件","impact":"CMC 对冲腿采集臂停摆"}），
   requested_at 2026-10-30 04:16:44Z——「缺凭据→credential/cmc 行」
   端到端产面实击发即在账。
   「补凭据→行 fulfilled（人工 UPDATE 口径可）」腿：Resolve 径
   （UPDATE…WHERE state='pending'→fulfilled+resolved_at+note）
   桩测结案语义实证；产面凭据三件今已在账（app_secrets
   cmc.email/cmc.password 键名列在案+app_config
   cmc.account_id=38941616），行仍 pending——人工 Resolve 未行
   属运维跟进面，机制锚无违（票面自钦「人工 UPDATE 口径可，
   自动关闭后续票」）。
3. 锚③ 门禁绿——达，双闸本席实跑：
   `scripts/check-env-reads.sh` → OK（硬拒/app_secret 域零直读，
   白名单 30 键）；`scripts/check_rejects_keys.sh` → OK（全仓
   Inc 键 ⊆ 注册集零漏；INFO 级存量动态 Inc internal/risk/
   check.go:148 非本票新增面）。

注记（报管理席裁决，不卡判）：
① 施工面 3 第二接点（fin_venue_untradeable/venue 凭据检查同族位）
   与施工面 4（ops face 只读查询面）均未落位（回填自白随后续票，
   仅 cmcArm 一接点在案）——验收锚未覆盖该两项故不违锚，锚覆盖面
   与施工面口径差报管理席裁决是否立追踪票；
② fin_0072_condition_gap.sql 头注残留「fin_0071」字样（0071→0072
   让位返修未刷注释）——纯注释零语义触，存案；
③ trx.go:160 `_ = condreq.Request(...)` 吞请求写败——孤儿日志
   先行发可观测性不失，trx.go 不在 D-508 热路径清单，模态存案；
④ 产面凭据已在账而 pending 行未 Resolve——业主响应径本职的
   人工跟进面（票面钦定自动关闭后续票），报管理席知悉。

判决：审计通过。

## 结果回填（返修——迁移号让位）

- fin_0072_condition_gap.sql：0071 已被 T-0173 host_metrics 占——
  让位 0072 重编（七断言过零撞号）；钦定列谱+UNIQUE pending 局部唯一+
  fin_data_retention 入册 permanent+domain.tsv fin 域入册。
- internal/platform/condreq：Request upsert+Resolve 径；cmc 凭据缺
  让位→credential/cmc 行接点；桩 TestRequestUpsertResolve 绿。
- 合并 master 冲突零（仅 backup_pg 系并合——异面零触）。

## 结果回填

- fin_0071_condition_gap.sql：钦定列谱+UNIQUE(venue,surface,kind) WHERE
  pending 局部唯一+fin_data_retention 入册 permanent——迁移闸全绿。
- internal/platform/condreq：Request upsert（pending 撞同键=刷新
  requested_at+payload 非重行）+Resolve 结案径（pending 才动防重复）。
- cmc 接点：trx.go 凭据键缺让位→condreq.Request credential/cmc 行
  （孤儿日志→请求盒）。
- 桩锚：TestRequestUpsertResolve（同键单行+fulfilled 结案）绿；
  check_migrations/check-env-reads/check_rejects_keys 全绿。
- 未接面：fin_venue_untradeable/ops face 读件随后续票（票面允许最小两接点）。
