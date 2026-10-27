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

- [x] 立单
- [x] 施工
- [x] 验收投产

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
