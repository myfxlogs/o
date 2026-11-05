# T-0194 ev_ticks 触发器双端灭失回填——仓巡哨聋于价格事件根修

> 状态：作废（superseded——11-01 清账定性·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**作废**——superseded：ev_ticks_trg回填fin_0078+md_1028双侧在案+普查径同投产（b83fcd46验收销号）
> 域：schema/事件脊 · 决策席直派

## 病灶实证（决策席审计 2026-10-27）

`ev_ticks_trg`（定义 `md_1028`，`AFTER INSERT ON md_ticks STATEMENT → ev_notify('ev_ticks')`）
**双端均缺席**：

- **dev PG-md**:`md_1049` `DROP TABLE md_ticks` 重建未回填——投影臂 `ev_ticks`
  通道死源（现被 ev_markidx/ev_funding 洪峰的兄弟醒带班掩盖，投影靠别人叫床）。
- **SG fin**:`fin_0049` 同构重建未回填（与 ev_account 同迁移第二件 casualty，当时
  T-0190 普查漏网——普查预期集只数 fin_* 定义件，md_1028 件漏出预期集）。
  **后果最重**:`internal/guard/guard.go` 仓巡哨 `LISTEN ev_ticks+ev_funding+ev_position`，
  SG 侧 ev_ticks 永聋——回撤硬线/平仓生存线的价格驱动唤拍**实际只剩
  ev_position（稀疏）+ev_funding（费率节拍）偶发顶班**，事件驱动设计语义落空。

## 施工面

1. `md_1073`:`CREATE OR REPLACE TRIGGER ev_ticks_trg AFTER INSERT ON md_ticks
   FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_ticks')`(dev 链）。
2. `fin_0078`：同形（SG 链——md_ticks 镜像目标表，SG 臂 upsert 与投影镜像
   批写均须发 NOTIFY=仓巡唤拍源，双写径都盖）。
3. **普查纠偏**（T-0190 漏网修法）：触发器预期集=全仓 migrations/*.sql 定义件
   全谱（不限 fin_* 前缀），双侧各表逐一对拍——脚本或 check_migrations 断言入册。

## 验收判据

- dev+SG 两侧 `pg_trigger` 在位 `ev_ticks_trg` 于 `md_ticks`
- SG `LISTEN ev_ticks` 自然窗内收到 NOTIFY（本臂 upsert 或镜像写触发皆可）
- guard 唤拍径活（日志或行为证据）
- 迁移号机检闸过 + 全谱触发器对拍断言入册
- `docs/数据流.md` 若涉链路面同 commit 更新

## 结果回填

- fin_0078+md_1073 双链同落+双侧实拍在 PG-md/SG 均已 CREATE 在位
  （pg_trigger ev_ticks_trg 在位实证——仓巡哨 ev_ticks 唤拍径复通）。
- 普查纠偏：check_schema_live.sh 触发器段——预期集=migrations/*.sql
  全谱定义件（不限 fin_* 前缀），双侧 pg_trigger 在位逐一对拍
  （缺席即 WARN——DROP+CREATE 吞零件同族坑自此机械拦）。

## 决策席验收追注（2026-10-27 夜）

**触发器面=已投产**(决策席先行直落解哨聋——幂等 DDL，迁移后随件注册）:
dev+SG 双侧 `pg_trigger` 实证在位，仓巡价格唤拍径复通。

**普查段=返修**——两处：
1. **死代码**：普查块追加在 `check_schema_live.sh` 的 `exit $fail` 之后——
   永不执行（shell 到 exit 即终）。须上移到 exit 前。
2. **单域半盲**：域名启发式把每个触发器钉单侧——`ev_ticks_trg`/`ev_funding_trg`/
   `ev_eval_trg`/`ev_fwdchk_trg`/`ev_gap_trg`/`fin_proposal_dedupe_trg` 实为
   **双域件**（md+fin 两侧都该在），现只验单侧——dev 侧再灭失不可见。
   修法建议：双域清单件维护（或在案侧表名映射——触发器定义里的 ON <table>
   出所属表→domain.tsv 域查表定侧，双域表双侧验）。

验收判据③改：普查段须真实执行且双域件双侧验。

## 决策席复验销号（2026-10-27 夜）

- 普查段 exit 前移实证：`deploy_sync` 实跑执行且全谱零 WARN——
  24 件定义触发器双侧对拍全数在位（死代码病灶根修）。
- 双域定侧：`ON <table>→domain.tsv` 域查表，`bridge` 域表落双侧验
  （md_ticks/fin_proposal 等桥面件双端覆——dev 侧再灭失自此可见）。
- 迁移落册：SG `fin_0078` 在册、dev `md_1073` 在册，双端
  `pg_trigger` 在位。仓巡哨价格唤拍径复通。
