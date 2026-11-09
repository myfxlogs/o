# T-0506 ops_event 写面语义闸——(seat,event) 合法性矩阵落库执行绝根修

- 状态：已批（业主令「从根因解决，下次不会再出现」）
- 级别：P2 机械轨写面完整面——T-0499 键形闸的同族姊妹件

## 病灶（2026-11-09 实证在案）

- COS 清账批 seat='COS' 写派单 5 票（T-0443/0479/0481/0482/0484）→
  `ops_ticket_phase` 视图照翻 dispatched（映射不看 seat），而
  `dm_queue.sh` 补拉候选滤 `latest_seat IN ('决策席','CTO')`——
  票呈「已派」而拉取窗全盲 **4h 零拉起**；
- 根因断层：相位投影 seat 宽 × 消费面 seat 窄，写面**零校验**——
  键形闸（T-0499 ops_event_key_gate.sh）只管 ticket 形，不管
  (seat,event) 组合合法性；裸 psql INSERT 更是一切闸全绕；
- 经验条目在案：dispatch-seat-whitelist-phase-lie。

## 范围（depgate 清单）

1. `migrations/`——新迁移件建 (seat,event) 白名单约束/trigger
   （只加新迁移不改存量件）；
2. `internal/store/pgstore/migrate_table.go`——登记新迁移号；
3. `scripts/meta/`——写径接入面（闸若有 helper 调用点的接线，
   以及存量组合枚举对拍桩）；
4. `scripts/meta/tests/`——语义闸桩测新件（拒写/放行/说明类
   宽容/存量组合全覆盖断言）。

## 判据

1. **合法性矩阵落库执行**：新迁移建 (seat,event) 白名单约束面——
   推荐 BEFORE INSERT trigger/约束表层（拒裸 psql，不靠写径自觉）；
   矩阵内容=现行钦定语义枚举（派单∈决策席/CTO、裁决∈决策席/CTO/
   裁定席/CEO/外审席、领单∈施工席族…），枚举依据=消费面既有 SQL
   判据逐条对拍成表，**不发明新语义**；
2. 违规写=loud 拒（ERROR 明文 seat/event/ticket 三元）——
   非 phase 驱动词（升件/处置/候水/送达等说明类）宽容放行，
   先钉「相位驱动词×seat」子集，全量收紧留观察窗；
3. 桩测：seat='COS'+派单→拒、seat='决策席'+派单→过、
   说明类事件任意 seat→过、历史合法写径零回归；
4. 机械轨/脚本写径全量过闸不炸（闸上线即真跑 mech_tick 一拍零拒——
   矩阵漏枚举=自残，桩测须先枚举在产全部 (seat,event) 存量组合
   `SELECT DISTINCT seat,event FROM ops_event` 对拍矩阵全覆盖）；
5. 全谱零新增 FAIL/SKIP。

## 禁线

- 矩阵以「消费面判据+存量组合」为准绳，禁凭直觉添删 seat；
- 不碰钱径；不改相位映射表本身（映射=钦定面）；
- 说明类事件（非相位驱动）不设闸——闸只罩会骗相位的写。

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）

### 实现摘要

- 新迁移 `migrations/fin_0154_ops_event_sem_gate.sql`：
  - `ops_event_sem_matrix`——(event,seat,note 条件) 白名单矩阵表，
    唯一键 (event,seat,note)；`domain.tsv` 册 `fin` 域；
  - `ops_event_sem_gate()` + `ops_event_sem_gate_trg` BEFORE INSERT
    触发器——相位驱动词 8 词（派单/裁决/领单/已批/起草/拉起/呈批/
    驳回）× seat 白名单落库执行；说明类事件（升件/处置/候水/送达/
    退班/领号/等锁/先例适用/部署/诊断草案/议题 等）零闸全放行；
    非法组合 `RAISE EXCEPTION` 明文 seat/event/ticket 三元 loud 拒；
  - 与 `fin_0121` hash 链触发器共存（BEFORE INSERT 双触发器同序链，
    C5 桩实证 hash 戳照落）；
  - `cos-relay` 条件例外：COS 族（COS/COS上行/席O-COS）写派单且
    note 带 cos-relay 标=历史合法中继形态放行，裸写必拒（存量 5 票
    病灶型即裸写）；
- `migrate_table.go` 登记 `{154,"fin_0154_ops_event_sem_gate.sql"}`；
- 桩测 `scripts/meta/tests/ops_event_sem_gate_test.sh`：
  A 静态矩阵自检（DDL/触发器/loud 三元/关键组合在案）→
  B 消费面对拍（dm_queue 派单席集、ceo_queue 呈批席集 14 席逐字，
  存量 `SELECT DISTINCT seat,event` 全枚举 247 组合分类
  ALLOW32/COND3/DENY18/UNGATED194 零漏）→
  C 真实 PG 触发器（testdb 租约+临时 schema，fin_0121+fin_0154 同装）：
  COS+派单裸写拒、决策席/CTO+派单过、8 词合法组合全过、说明类任意
  seat 过、cos-relay 例外放行、DENY18 型抽查全拒、拒写零落账、
  hash 链共存；
- Go 夹具对齐（桩组合改合法语义席，非语义变更）：
  `cmd/core/cockpit_face_test.go`（席桩→施工席）、
  `cmd/core/pipeline_face_test.go`（合成席→施工席/无头施工席/
  决策席）、`cmd/core/watch_pipeline_test.go`（Dev→施工席族）、
  `internal/portfolio/hunger/drafter_test.go`（决策席领单→派单/
  施工席领单序）、`internal/store/pgstore/ops_precedent_test.go`
  （桩裁→证据席裁决）。

### 测试证据

- `sem_gate: PASS=76 FAIL=0`（A/B/C 三段全绿）；
- `go build ./...` + `go vet ./...` 全绿；
- `check_migrations.sh` OK（七断言过——新表 domain 册在案）；
- `scripts/check_all.sh` mode=full 全链绿，绿戳对口 HEAD
  `584caa0a2`（戳件 `.git/check-green-584caa0a2…`）。

### 偏离说明

- 无范围外改动。工单「scripts/meta/ 写径接入面」按 DB 触发器落地
  ——裸 psql/全部 helper/未来新写径同闸，无需 helper 接线点；
  存量组合枚举对拍桩即测试 B 段。
- rebase 带入 T-0507 后 ceo_queue 席集新增 MECH（层扫 digest 席），
  A3 对拍检出漂移即补 `('呈批','MECH')` 矩阵行——消费面判据随
  消费面漂移的实证，非凭直觉添席。
- 早前全谱见的 `TestReportMacroEventsCount` 失败系 T-0510 病灶，
  已由 merge `c3cb09093` 根修——本分支 rebase 后全绿，零遗留红，
  无需在本票处置。
