---
schema_version: 2
id: id-cursor-commit-order-skip
kind: pitfall
risk: high
status: verified
keywords: [cursor, watermark, id>, bigserial, commit-order, marten, skip-read, projector, ops_event, fin_push_ledger, param_intake, overlap-window]
scope: [projector, consumer, mirror, lifecycle, feishubridge, feed_push, ops_relay]
summary: bigserial/ts 序≠提交序——低位行事务未提交期间高位行先入水位=低位行永久跳读；修法=重叠窗复读(幂等upsert)/滞后余量/FIFO序，水位推进须连续成功
first_seen: 2026-11-23
last_seen: 2026-11-23
evidence_refs:
  - type: ticket
    ref: 工单O-30635（全仓投影/消费面增量游标普查——Marten V9.16.1 型）
  - type: commit
    ref: fix/cursor-skip-audit（重叠窗+滞后余量+连续水位三处修法同批）
---

## id>/ts> 增量游标按分配序推进=迟提交行永久跳读（Marten 型全类病灶）

- **机理**：PG bigserial/时间戳的「分配序≠提交序」——事务 A 取低 id
  未提交、事务 B 取高 id 先提交；消费环 `WHERE id>wm` 读到 B 推进
  wm 越过 A 的 id，A 随后提交=**永久静默跳读**（无报错无升级，
  Marten V9.16.1 同型）。跳读成立充要条件：低位行事务在水位推进
  时刻仍未提交。
- **本仓实证面（O-30635 普查 8 查询点/6 文件全修）**：
  - `internal/projector/ops.go` fin_proposal `id>propWm` +
    fin_opp_eval `ts>wm`——md 侧 research ~15 臂 goroutine 并发写，
    内存水位仅 boot 全量对账兜底→**重叠窗**（id>wm-500 /
    ts>wm-10min，uProposal/uEval 幂等 upsert 兜零代价，对齐
    replayTsTables 10min 惯例）。
  - `internal/research/analyst/analyst.go` `ORDER BY id DESC
    LIMIT 50`——**确定性跳读**（积压>50 时水位钉最新 50 底，中段
    行永丢，与竞态无关）→改 ASC FIFO 头吃。
  - `internal/portfolio/lifecycle/param_intake.go` ops_event
    `id>wm`（ops_state_row 落库水位**无重扫自愈**）——全仓最高
    并发写面→`ts<now()-60s` 滞后余量。
  - `cmd/feishubridge/main.go` fin_condition_gap `id>wm`（文件
    水位）——**发败水位连进 bug**：`continue` 让后位成功行
    writeWM 跳过败行=确定性丢卡（非竞态）→连续水位制（blocked
    闸+ b.sent 免重发）+`requested_at` 60s 滞后。
  - `cmd/core/feed_push.go` pending + `scripts/ops_relay.sh`×2
    ops_event `id>wm`——同滞后余量 60s。
- **安全判据（①不动面）**：同事务/同连接写读耦合、ts 水位带
  重叠窗复读（tsops 五表 replayTsTables `max(wm,now-10min)`
  先例在案）、全表/全 id 集重扫（regime.go/反向桥全族）、
  单写者序写（notify funding_settle=legfunding o-core 单实例）、
  窗重扫+push 键去重（notify opened/closed 族）。
- **不适用判据（③）**：`id>t.id`/`ts>e.ts` 关系谓词（后见关前/
  去重 join/窗内过滤）、LIMIT/OFFSET 分页、last_seen 行戳、
  xmin 哨兵、迁移静态校验——不驱动持久增量消费游标。
- **修法选型**：幂等 upsert 已在→**重叠窗优先**（spec 钦定）；
  重放有副作用（审计行/外发）→**滞后余量** `ts<now()-Δ`（残余=
  >Δ 长事务，量级权衡写进注释）；水位须**连续成功推进**——
  「发败 continue」让后位行跳号=确定性丢件，比竞态更常见。
- **排查捷径**：凡见 `WHERE id>$n`/`ts>$n` 先问三件——游标存哪
  （内存/文件/DB 行）、源表写者几个并发、跳读后有无重扫兜底。
  三问皆答才准判①。
