# T-0190 ev_account 触发器随表重建灭失——事件脊断肋回填

> 立单：决策席 2026-10-27（SG「REST 已死」根查）
> 域：migrations fin 链 + SG o-postgres
> 量级：微件（一行迁移+在位/定义对拍普查断言）

## 病灶实证（决策席取证在案）

```
migrations/fin_0028：ev_account_trg AFTER INSERT ON md_account_events 定义在册
migrations/fin_0049:32：DROP TABLE md_account_events; CREATE TABLE（触发器随毁）
SG pg_trigger 现拍：ev_* 在册 9 件，ev_account_trg 缺席（唯一缺）
```

**因果链**：okxpriv 私有帧 INSERT md_account_events→零 NOTIFY→
ledger scan（监听 ev_cashflow/ev_account/ev_position）只剩稀疏
cashflow/position 事件唤醒→`restOKAt`（对拍环尾端戳）间隙超
System 页 2min 死阈→「OKX REST 已死 4 分钟」假死相。

**连带影响**：scan 欠驱动=对账环节拍实质变慢（事件密度降一档），
holding/sync（同听 ev_account）同病——私有帧再鲜也叫不醒它们。

**同族病灶**：T-0187 fin_position 引用缺席姊妹——DROP+CREATE
迁移吞零件（表重建=触发器/索引/授权全毁，重建面须零件清单对拍）。

## 修法钦定

新迁移 `fin_0077_ev_account_trg_restore.sql`：

```sql
CREATE OR REPLACE TRIGGER ev_account_trg AFTER INSERT ON md_account_events
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_account');
```

并**普查断言**：`pg_trigger` 在位集 vs migrations 定义集全对拍——
其余 DROP+CREATE 迁移（fin_0049/md_1049 同族件）涉及表的触发器
逐件核对（本票已查 ev_* 族，施工面再核非 ev_ 前缀触发器与
索引/授权类零件无漏）。

## 验收锚

1. SG `pg_trigger` 见 `ev_account_trg` 在位；
2. 私有帧 INSERT→ledger scan 被唤醒（可插测试帧验 NOTIFY 链路，
   或观察 restOKAt 间隙不再>2min）；
3. System 页 REST 由红/黄复绿并稳定（不再 flap）；
4. 触发器在位/定义对拍清单留票面（非 ev_ 族无缺席实证）。

## 状态

审计通过
审计席：headless @ 2026-10-30

## 结果回填（SG 实拍+迁移同落）

- fin_0077_ev_account_trg_restore.sql：ev_account_trg AFTER INSERT 回填
  ——SG 实拍已落（pg_trigger 见 ev_account_trg 在位+schema_migrations
  version=77 在册）；私有帧 NOTIFY 链路复通。
- 触发器在位/定义对拍普查：SG pg_trigger 19 件全数对拍——18 件
  migrations 定义在册，唯一 metadata_insert_trigger 系
  _timescaledb_catalog.metadata 内部件（扩展所有非迁移面=如实
  注明非漏件）；非 ev_ 族（app_config/secrets/active_set/dedupe/
  class_stamp）全在册零缺。


## 决策席验收锚回执（2026-10-27）

1. ✅ SG pg_trigger 见 ev_account_trg 在位（pg_get_triggerdef 与
   fin_0028 原本逐字同形：AFTER INSERT FOR EACH STATEMENT
   EXECUTE FUNCTION ev_notify('ev_account')）；
2. ✅ NOTIFY 功能实证：LISTEN ev_account 12s 窗内自然帧触发 3 次
   （私有 account 帧秒级流入——事件脊肋骨复接非纸面在位）；
3. ✅ fin_0077 在册（schema_migrations version=77，applied
   2026-10-27 10:49Z）；deploy_sync 漂移哨兵 fin 链 live=77 ≥ 登记=77；
4. ✅ 全席 beat 8s 鲜（司账/私流守/核账 等 8 席——事件环全链跑）；
5. ✅ 19 件触发器普查留票面（施工回填在案）。

销号投产。

## 判决注记（无头审计逐锚对账——证据=收口 commit 66f9182 全量 diff
##   + SG o-postgres 只读实拍 + Overview RPC/LISTEN 亲测 @ 2026-10-30 12:47~13:23Z）

- **锚①（SG pg_trigger 见 ev_account_trg 在位）——过**：审计席
  独立实拍 SG `o` 库——`ev_account_trg` ON `md_account_events`
  在位且 tgenabled='O'（启用态非摆设）；pg_get_triggerdef=
  `CREATE TRIGGER ev_account_trg AFTER INSERT ON public.md_account_events
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_account')`，
  与 fin_0028:16 原本逐字同形。产物三证：migrations/
  fin_0077_ev_account_trg_restore.sql 在 master 实存（内容=钦定
  修法字面+注释头）+ migrate_table.go {77,fin_0077…} 登记 +
  schema_migrations version=77 applied 2026-10-27 10:49:41Z。
- **锚②（私有帧 INSERT→ledger scan 被唤醒）——过**：审计席亲测
  LISTEN ev_account 420s 窗（13:16~13:23Z）收自然帧 NOTIFY ~80 发，
  来自 5 个写者 PID——私有帧 upsert 语句实发通知，非纸面在位。
  下游三重活证：earn_interest 事件行→interest cashflow 两拍同毫
  （12:06:03.535→.538、13:08:00.794→.797=scan 内 postInterest
  写入）；account upsert 13:00:40→restOKAt 13:00:55（醒→对账→
  REST 实调回环）；码面对拍 ledger ListenLoop 纯 NOTIFY 驱动零
  轮询（internal/platform/events/events.go:110-150），订阅面
  =ev_cashflow/ev_account/ev_position（ledger/run.go:72-73），
  holding/sync.go:46 同订 ev_account——断肋复接且有消费端。
  （审计过程注记：前两窗以 `psql -c "a;b"` 同隐式事务跑致
  LISTEN 提交方生效=自聋窗，依法改用 stdin 分行即收证——
  取证法错非链断，如实存案。）
- **锚③（System 页 REST 由红/黄复绿并稳定）——过**：CoreService/
  Overview 三样本跨 ~13min 复采（12:48/12:50/13:00Z）——okx
  REST 通道 state=0 绿且 lastTs 逐样本秒级推进（12:48:15Z→
  12:50:15Z→13:00:55Z=对拍环每拍实调成功续戳，非僵拍）；
  同面公开 WS/私有 WS/MT5 户全绿。ops_event 48h 零 REST 死警、
  o-core 24h 日志零「已死」——稳绿不 flap。
- **锚④（在位/定义对拍清单留票面+非 ev_ 族零缺席）——过**：票面
  『结果回填』段普查清单在案。审计席独立复拍：SG `o` 库 public
  非内触发器现 24 件，逐件回映迁移定义——全部由已应用 fin 链
  （schema_migrations 1-90 连续在册）定义或同族回填件覆盖，零
  无源件；非 ev_ 族（app_config/app_secrets/active_set/class_stamp/
  dedupe/untradeable/arch_eval_reject/ops_*）全在册零缺。重建四表
  （fin_0049）索引面实核：md_account_events/md_ticks/md_mark_index
  PK 各 1+md_dcd_product PK+uly 索引 2 件，与迁移定义符。
- **对拍口径注记（如实存案，不构驳回）**：迁移定义集有而 SG 缺席
  者全为 md_1xxx 链独有件（arch_eval_trg/arch_signal_trg/ev_dcd_trg/
  ev_markidx_trg/ev_proposal_md_trg/ev_signal_trg）——schema_migrations
  ≥1000 零行=md 链从未在 SG fin 库落号（md_* 表系 fin_0049 等
  fin 链镜建），缺席属链界正常非漏件；ev_funding/ev_fwdchk/
  ev_gap 三件在位仅 md_1028/1045 定义，md_1028 注释自钉
  「SG 侧历史同在」=分链前历史件幸存。
- **禁线核查——未违**：收口 diff 三件=迁移 SQL+登记表+票面回填，
  零触 venue/钱径/部署面；CREATE OR REPLACE 幂等形在钦定修法
  字面内。同族第二 casualty ev_ticks_trg 已由姊妹票 fin_0078
  回填在册（T-0194 链）。
