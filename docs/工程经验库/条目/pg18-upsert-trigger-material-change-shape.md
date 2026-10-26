---
schema_version: 2
id: pg18-upsert-trigger-material-change-shape
kind: pitfall
risk: medium
status: verified
keywords: [PG18, transition table, REFERENCING, ON CONFLICT DO UPDATE, row-level 触发器, TG_OP, NOTIFY 洪泛, 实质变才发, 单事件限制]
scope: [migrations, 一切想给 upsert 高频表挂触发器收敛 NOTIFY/审计/级联的面]
summary: PG18 statement 级 transition table 触发器禁多事件（INSERT OR UPDATE 合一即 DDL 报错）+INSERT 单事件触发器禁声明 OLD TABLE——upsert（INSERT..ON CONFLICT DO UPDATE）写拍在 statement 级无法区分净新增与覆写；可行形=row-level 多事件触发器（DO UPDATE 臂 TG_OP='UPDATE' 且 OLD/NEW 可比、DO NOTHING 臂零 fire）+同事务空 payload 同值去重兜批合流
first_seen: 2026-10-26
last_verified: 2026-10-26
evidence: [T-0162, PG 18.6 t0162 探针, migrations/0064_notify_flood_shrink.sql]
---

# pg18-upsert-trigger-material-change-shape——upsert 高频表「实质变才发」触发器形状律（PG18 实证）

## 需求形
upsert 高频表（`INSERT..ON CONFLICT DO UPDATE` 逐行 autocommit 写拍）想在
**不动写方**的前提下把 NOTIFY 收敛到「实质变才发」（同值覆写静默）。
第一反应=statement 级触发器+transition table 批判 OLD/NEW——**在 PG18 走不通**。

## 三条硬律（PG 18.6 实证，t0162 探针）
1. **transition table 触发器单事件**：`AFTER INSERT OR UPDATE ... REFERENCING
   OLD TABLE AS old_table NEW TABLE AS new_table` → DDL 直接报错
   `transition tables cannot be specified for triggers with more than one event`。
2. **INSERT 单事件触发器禁声明 OLD TABLE**：`OLD TABLE can only be specified
   for a DELETE or UPDATE trigger`——而 upsert 全程只 fire INSERT 类触发器，
   statement 级**拿不到被覆写行的旧版**，净新增 vs 覆写不可分。
3. **row-level 多事件触发器才是可行形**（`FOR EACH ROW` 无单事件限制）：
   - 语句形=普通 INSERT → 该行 `TG_OP='INSERT'`，OLD 无值（取之=NULL 不炸）；
   - 语句形=upsert 且落 **DO UPDATE 臂** → 该行按 **`TG_OP='UPDATE'`** fire
     且 **OLD/NEW 完整可比**（old_v/new_v 实拍验证）；
   - 落 **DO NOTHING 臂** → 该行**完全不 fire**（天然零噪）。
   - 批合流不用触发器做：row-level 每行一次 `pg_notify(ch,'')`，同事务内
     空 payload 同值由 PG 天然去重（0028 律）→ 一事务一醒。

## 落地样例
见 `migrations/0064_notify_flood_shrink.sql`（ev_notify_eval：TG_OP='INSERT'
恒发；否则 verdict/gross_apr/net_apr/legs/evidence 任一 `IS DISTINCT FROM`
才 notify；ts 是写拍元数据不计）。

## 教训
- 给高频写拍表挂触发器，先探语句形谱（普通 INSERT/DO UPDATE/DO NOTHING/
  普通 UPDATE）再定 statement vs row-level——**transition table 的能力面
  比文档直觉窄**（单事件+INSERT 无 OLD）。
- 「每行一 NOTIFY」的洪泛根因常是**逐行 autocommit**（每行独立事务，
  同事务去重永不生效），而值变率可能极低（T-0162 实测 600 行/拍仅 3 行动）
  ——先测值漂移率再选治法：漂移≈0 → 触发器实质变静默即降两个数量级；
  漂移高 → 才需要写方批事务合流。
- UP-SERT 触发器行为是版本敏感面（PG10 文档与实行为最易混处）——上生产
  前在目标版本上跑探针（建 scratch 表+RAISE NOTICE 逐形打 TG_OP/OLD/NEW）。
