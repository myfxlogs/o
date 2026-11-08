---
schema_version: 2
id: ops-event-ticket-key-hygiene-double-gate
kind: pitfall
risk: medium
status: verified
keywords: [ops_event, ticket键, 拼接伪键, 键卫生闸, ops_ticket_phase, 视图守卫, loud拒写, 多行拼接]
scope: [scripts/meta/ops_event_key_gate.sh, migrations/fin_0153_ops_ticket_phase_key_guard.sql, scripts/meta/* ops_event写径]
summary: ops_event.ticket 拼接伪键（空白/逗号族或多票号形态）须写面落库前拒+投影面守卫双闸——只修读径归一=新脏行继续产，相位永驻尸；loud=升件行落账非静默截断（截断造新伪键）
first_seen: 2026-11-09
last_verified: 2026-11-09
skills: []
evidence_refs:
  - type: ticket
    ref: T-0499（dm 事件写面 ticket 键卫生——多行拼接伪键根修）
  - type: code
    ref: migrations/fin_0153_ops_ticket_phase_key_guard.sql（视图 WHERE 守卫）
  - type: code
    ref: scripts/meta/ops_event_key_gate.sh（写面共享闸）
---

## 症状

ops_event.ticket 出现 `'T-0270\nT-0277'`、`'T-0276\nT-0276\nT-0276\nT-0276'`
形多行拼接值（dm-wake 批量 prompt 一行多票号经 grep -m1 -oE 无 head
提取原样灌库——根因见 grep-m1-o-multimatch-newline-tid）。一切按
ticket 精确匹配的下游面（相位投影/领单判重/deploy 挡件/dm_queue 候选）
永远查不到→pulled 相位永驻尸，2026-11-09 COS 清账实锤 5 个拼接键占
尸源大头。

## 根修（T-0499 双闸）

- **写面闸**：`scripts/meta/ops_event_key_gate.sh` 单一 helper——判据
  =含空白/逗号族（空格/\n/\t/，/）或 T-NNNN 形态≥2；rc1 loud 拒写
  （stderr+ops_event 升件行 ticket='' 落账，24h 同键去重）。接线面=
  scripts/meta 一切可变 ticket 写径（escalate/ops_esc 双漏斗最省——
  一切 esc() 调用族同被罩）。调用面 `-x` 判空 fail-open：闸缺席放行
  不阻写（OCI 单件部署无兄弟件先例；卫生面非链环）。
- **投影面守卫**：新迁移给 ops_ticket_phase 视图 WHERE 加
  `ticket !~ '[[:space:],，]'` +`ticket !~ 'T-[0-9]{4}.*T-[0-9]{4}'`
  ——存量脏行 append-only 不清洗，读面守卫归位（同
  split_part 归一律：脏不修由读面挡）。
- **多行 INSERT...SELECT 写径**（dm_autoapprove 批件保鲜）：预扫票列
  过闸→净键 IN 表→原子 INSERT...SELECT 内嵌 stale 条件重估——相位
  竞态语义不拆事务，闸判据单源不散进 SQL 正则（防双写漂移）。

## 教训

- 键卫生是**写面职责**不是下游各自防——凡以拼接产 ticket 的径同病；
  「落库前拒」的 loud 形=升件行（ticket='' 不入票域）非静默截断。
- sed 按行读 \n 不入模式空间、tr 按字节切多字节字符——键预览归一
  用 bash 参数展开 `${v//$'\n'/·}` 最稳。
- view 迁移只加新迁移不改存量件；CREATE OR REPLACE 保权但条件
  GRANT 块仍幂等重列（grant-not-in-migration-silent-dead 同律——
  链内自足防授权面断档）。

## 关联

- grep-m1-o-multimatch-newline-tid（根因条目——本件=写面+投影面根修）
- grant-not-in-migration-silent-dead（迁移 GRANT 纪律）
- T-0499 票面（判据四条+禁线：不改相位映射语义/不批量 UPDATE 历史行）
