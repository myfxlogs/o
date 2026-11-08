# T-0499 dm 事件写面 ticket 键卫生——多行拼接伪键根修

- 状态：待验（原批：业主授权 COS 代位裁——积压清账逮出的写面病灶）
- 级别：P3 机械轨卫生面

## 病灶（实证在案）

`ops_ticket_phase` 里活着一批 pulled 相位永驻尸——其 `ticket` 键是**多行
字符串拼接值**（形如 `'T-0270\nT-0277'`、`'T-0276\nT-0276\nT-0276\nT-0276'`）：

- dm-wake/dm 写事件径曾把**多个票号塞进同一个 ticket 字段**落库——
  下游一切按 ticket 精确匹配的查询面（相位投影/领单判重/deploy 挡件/
  dm_queue 候选）永远查不到它，相位永驻；
- 2026-11-09 COS 清账实锤：pulled 尸族里 5 个拼接键占尸源大头，
  已按 SELECT chr(10) 特征批量 作废 销账（事件行在案）；
- 同类隐患面：凡以空格/换行/逗号拼接写 ticket 的径同受此病——
  写面该在落库前拒，不该靠下游各自防。

## 范围（CTO 钦定回填——depgate 缺清单补位）

1. `scripts/meta/`——ops_event INSERT 写面键卫生闸（dm_ledger/dm_queue/
   dm-wake 族一切写径入口共享闸件，宜单一 helper 收口非逐件散改）；
2. `migrations/`——新迁移件给 ops_ticket_phase 视图 WHERE 加 chr(10)
   守卫（投影面清尸——只加新迁移不改存量件）；
3. `scripts/meta/tests/`——键卫生桩测（拼接伪键拒写 loud+合法单键不受影响）。

## 判据

1. `ops_event` 写径（dm_ledger/dm_queue/dm-wake 族一切 INSERT 面）对
   ticket 字段做**键卫生闸**：含空白字符（空格/\n/\t）或多票号形态=
   loud 拒写（非静默截断——截断会造新伪键）；
2. `ops_ticket_phase` 视图或消费面对存量拼接键不再产生相位行
   （视图 WHERE 加 ticket 不含 chr(10) 守卫=投影面一并清）；
3. 桩测：人造「T-XXXX\nT-YYYY」写入→拒写 loud、视图零相位行；
   合法单键写入不受影响；
4. 全谱零新增 FAIL/SKIP。

## 禁线

- 不碰钱径；不改相位映射语义本身（映射表是钦定面）；
- 拼接键的历史账已作废销——本票只修写面+投影守卫，
  不做批量 UPDATE 改写历史行（台账不可篡改面）。

## 结果回填

- 状态：待验
- 实现摘要：
  1. **写面共享闸** `scripts/meta/ops_event_key_gate.sh`（新件）——
     伪键判据=含空白/逗号族（空格/\n/\t/，/）或 T-NNNN 形态≥2；
     rc0 净放行（空键合法）/rc1 loud 拒写（stderr+ops_event 升件行
     ticket='' 落账，note 前缀「ticket键卫生拒写 <ctx> h=<md5-8>」
     24h 同键去重防刷屏）。闸自身写径全尽力而为不反噬调用面；
     调用面 `-x` 判空 fail-open（OCI 单件部署无兄弟件先例在案）。
  2. **接线面**（diff ⊆ 票面涉及面——scripts/meta 一切可变 ticket
     写径）：escalate.sh/ops_esc.sh 双漏斗（覆盖全部 esc() 调用族）、
     gauges_tick 量表名、dm_autoapprove 候选集+批件保鲜（stale
     INSERT...SELECT 改「预扫票列过闸→IN 表原子重估」保相位竞态
     语义）、dm_depgate 在审/候选双集+insert_dispatch_event、
     dm_ledger 补登、dm_suspend/mech_tick/dispatch_water 拉起候水、
     verdict_reflow 归流标记、case_writeback 归流行。字面票键写径
     （''/'T-0466'/'META-*'）不在可变面未接。
  3. **投影面守卫** `migrations/fin_0153_ops_ticket_phase_key_guard.sql`
     ——ops_ticket_phase 视图 CREATE OR REPLACE 重建，fin_0147 词集/
     相位映射一字不动，内层 WHERE 加 `ticket !~ '[[:space:],，]'`
     +`ticket !~ 'T-[0-9]{4}.*T-[0-9]{4}'`（chr(10)⊂[[:space:]]
     尸源全覆盖）；条件 GRANT 块同 fin_0147 幂等重列；
     migrate_table.go 登记 {153}。
  4. **桩测** `scripts/meta/tests/ticket_key_gate_test.sh`——A 闸单测
     12 断言（六形伪键拒+loud+去重/净键零写）、B escalate 接线抽查
     3 断言、C 真库视图守卫 7 断言（testdb_lease 径池空 skip）——
     全 23 断言绿。
- 自测：bash -n 全改动件绿；新桩测 23/23 绿（含真 PG 视图实证）；
  邻面回归 ops_esc/dm_autoapprove/dm_depgate/dispatch_water/
  mech_tick/verdict_reflow/case_writeback/gauges_sounding/
  dm_suspend/dm_ledger/dm_ledger_lineage 十件全 PASS；
  check_migrations 七断言绿、check_tolerate_report 绿、gofmt 净。
- 残留面（票外未动，呈决策裁）：report.sh（~/bin）与
  /opt/meta/bin/dm-spawn.sh 在仓外——拼接伪键最后产源径不在本票
  scripts/meta 域界；写面闸+视图守卫已使其产物不产生相位行，
  是否补仓外闸件由决策席裁。

## 返修1·先红双点证据（check_fix_red_first 驳回件补位）

- **修前红点**（对照 `origin/master=878e70032` 旧态真跑重放，非捏造）：
  - W 写面闸缺席径：旧 `escalate.sh`@origin/master（无闸）+psql 桩，
    伪键 `$'T-9020\nT-9021'` 入参→`rc=0` 且
    `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)`
    落库行在案=**FAIL（伪键落库无拒）**；
  - V 视图守卫缺席径：旧 `ops_ticket_phase`（fin_0147 版 DDL 原样
    抽取应用）+C 段同种子（3 合法+4 伪键形）→三条「伪键零相位行」
    断言全红：`空白族漏行=2`、`多票号漏行=4`、`chr(10) 漏行=1`、
    `相位行总数=7`（伪键 4 行全成相位：`T-9020\nT-9021→pulled`、
    `T-9022 T-9023→dispatched`、`T-9024,T-9025→pulled`、
    `T-9026|T-9027→pulled`）——修后同断言=0/0/0/总数 3。
- **修后绿点**：本支 `875004fb9`（rebase 后修复 commit）同断言桩测
  `ticket_key_gate_test.sh` 全绿 `PASS=23 FAIL=0`（A 闸 12+B 接线 3+
  C 真库视图 7，testdb_lease 真 PG 实证）——红绿同断言同种子双点在案。
- **域界呈裁结论**（决策席已裁，回填在案）：report.sh（~/bin）与
  /opt/meta/bin/dm-spawn.sh 仓外产源径**不立票补闸**——fin_0153
  投影守卫结构性兜住一切写源伪键（不产生相位行），域外补闸收益
  边际；观察项记入残留面（上节）即可。
