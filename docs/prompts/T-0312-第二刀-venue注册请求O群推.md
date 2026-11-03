# 施工提示词：T-0312 第二刀——venue 注册请求 O 群推

**角色**：`[角色:施工]`　**票**：T-0312 ④件（票面范围补充节已落）
**工作树**：`/opt/o-wt/t0312`　**分支**：`t/0312-regime-push`（同支续提）

## 背景

scout `maybeRecommend` 达阈后台账翻 `state='recommended'`（OCI md 正本→
projector 镜像 SG 已通），但 OCI 侧 alerter 是 `NewAlerter(nil,...)`——
sender 缺席 fail-closed 只落 journal，「请注册」永远到不了业主。O 群推送
通道在 SG o-core。**本件补最后一里：SG 侧盯镜像翻牌→O 群推**。

## 范围（四件，照 fin_0100/regime.go/venue_cand.go 先例逐字对形）

### ①迁移 `migrations/fin_0101_venue_req_push.sql`

幂等锚表（fin 域，写者=o-core 本体池——`md_regime_push` 同款处理）：

```sql
CREATE TABLE IF NOT EXISTS md_venue_req_push (
    venue      text        NOT NULL,
    request_at timestamptz NOT NULL,
    pushed_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (venue, request_at)
);
```

- 锚键=（venue,request_at）**对**非 venue 单键——30d 重发闸下同 venue
  可二次请求，每次 last_request_at 不同须各推一次。
- `domain.tsv` 落 `md_venue_req_push	fin`（对齐 `md_regime_push	fin` 行）。
- 注释/约束审计标记行照 fin_0100 同款（`('md_venue_req_push','state',…)`）。

### ②projector `internal/projector/venue_cand.go`

`projectVenueCandidates` batchExec 成功后：统计批内 `state=='recommended'`
行数，>0 → `r.sg.Exec(ctx, "SELECT pg_notify('ev_venue_req','')")`——
`regime.go` R2 同款空批不唤（禁无条件每批都 notify）。

### ③推送件 `internal/platform/notify/venue_req.go`

照 `notify/regime.go` 形新件。Scan 逻辑：

```sql
SELECT c.venue, c.last_request_at, c.metrics
  FROM fin_venue_candidate c
  LEFT JOIN md_venue_req_push p
    ON p.venue=c.venue AND p.request_at=c.last_request_at
 WHERE c.state='recommended' AND c.last_request_at IS NOT NULL
   AND p.venue IS NULL
```

消息组面：metrics jsonb 的 `boards[]` 里取达阈族（`net_apr_mid>=baseline+margin`
且 persist 最高者），组中文一条：

```
【venue-scout】请注册 {venue}：{family} 30d 实测净年化 top-{n} 中位 {x}% vs
基线 {y}%（持续占比 {p}%）——红线件：新平台注册归业主裁决。凭据到手后落
skills/venue-scout-{venue} 能力账径。
```

boards 解析败/空 → 退保底文「请注册 {venue}（state→recommended，度量件查
fin_venue_candidate.metrics）」——**禁因组文败丢推送**。

- 发成功→INSERT 锚；**发败不落锚**（下拍重试）；关态→落锚不发（regime 同语义）。
- Sender/chat_id/on 三个注入面照 regime.go 同款装配。

### ④`cmd/core/event_push.go` 挂 LISTEN

`events.ListenLoop(ctx, pool, "venue_req_push", []string{"ev_venue_req"}, vq.Scan)`
+`notify.feishu.venue_req.enabled` 默认开——`ev_regime` 挂法逐字同款。

## 禁线

- 不改 OCI 侧 `NewAlerter(nil,...)`（骨架豁免在案，SG 推即正解）。
- 不动 fin_venue_candidate 表结构（metrics jsonb 已带 boards，够用）。
- 锚表只 o-core 写——projector 侧零锚表写径。
- 消息红线句钉死（新平台注册=红线三件归业主，机器只请不催办）。

## 验收（决策席亲跑三重硬迹，缺一项不审）

1. OCI md 手工 `UPDATE fin_venue_candidate SET state='recommended',
   last_request_at=now() WHERE venue='mexc'`→projector 镜像→O 群实收
   「请注册 mexc」；
2. o-core 重启后 `md_venue_req_push` 锚数不变、零重复推送；
3. 验后还原 `state='measuring'`+清锚行。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 锚键若 venue 单键，30d 重发闸下二次请求被吞 | 语义坑 | 锚键改（venue,request_at）对——范围①已钉 |
| 2 | metrics jsonb boards 字段名未实测过形 | 偏差险 | 保底文案兜底+推送不丢——范围③已钉 |
| 3 | 手工试发会污染真实台账态 | 验收残留 | 验收第3条钉还原+清锚 |
| 4 | notify 空批次每拍空跑 Scan | 小疵（R2 前科） | 范围②钉 recommended>0 才唤 |
