# T-0231 Phase-1 桥面收口——bridge_outbox 双车道 + 自包含身份 + admission 独立表

> 状态：待施工 | 优先级：P1（D-0036 终裁 §12.5 序6——待终切三锚绿后启）
> 立项：决策.终席 @ 2026-10-28（D-0036 定稿§12.3 Phase 1 钦定）
> 前置：终切 serve 三锚绿（T+72h）+ T-0223 存在性列在场
> 依据：docs/decisions/D-0036 §3-P0-2/P1-1/P1-2、§12.2 八问裁2/3/5

## 病灶

事实类消息走 `LISTEN/NOTIFY`（易失）+ 时间戳游标 + 表镜像——
终席亲证实害：md 提案 405143 落库 10min 桥不动，手写 NOTIFY 一推
即通（NOTIFY 非持久+`mdWake("")` boot 不跑提案投影=漏帧即卡）。
eval id 译码（evalmap.go 自然键→max(id)+1+setval 序列守卫）=
跨库伪外键，T-0211 已爆 509 行同键异 id。
`now()-10min` 对账夹逼=陈漏容忍带（reconcile.go:191-200）。

## 钦定修法（终裁 §12.2 裁2/3 定版）

**双车道分制**：
- **事实道**（proposal/eval/signal/dcd catalog/macro/config/roster）：
  `bridge_outbox(seq bigserial, kind, key, payload jsonb, created_at)`
  ——写径同事务触发器追加；消费端按 per-kind 游标 `seq>ack` 拉取，
  幂等 apply 提交后写 `bridge_ack(kind, seq)`，批级 ack。
  **跨 kind 序契约=自包含载荷推到底**：proposal 消息内嵌其引用
  eval 的完整快照——消费契约零依赖序，天然幂等（终裁钦定，禁
  依赖拉取序）。
- **ts 道**（ticks/mark/funding 活跃集）：**保留最新值镜像+持久化
  水印**（重启不扫全表），**删 `now()-10min` 夹逼**——落后即对账，
  不留陈漏容忍带。

**身份换锚**：提案/eval 桥载 `(origin, origin_id)` 复合身份；
`eval_uid` 自然键贯穿；**evalmap.go 译码器+max(id)+1 分配+setval
序列守卫三件同 commit 删**（遗产行一次性 remap 迁移）。
**admission 独立表**：提案载荷 md 正本 append-only；准入态 SG
正本独立表（字段级双主取销）；SG→md 回程运输方式票面点名
（册 diff vs 对称 outbox 二选一）+回程时延锚。

**顺序**：outbox 双写（旧径保留）→ 对拍 → 切读 → 删旧径。
lag 指标（`max(seq)-ack`）入健康板（终裁 §12.2 裁5 机检族）。
outbox 分区/保留/清理策略先定（防膨胀）。

## 验收锚（D-0036 §8 P1 全项）

- 杀 projector 中途零丢零重（混沌测试：apply 中杀→重起重放幂等）。
- 断线 30min 恢复追平；重启不全表扫（持久水位实证）。
- `lag` 健康板可见；提案无 id 译码/序列守卫（grep evalmap/
  max(id)+1/setval 零引用）；同 uid 两侧对拍一致。
- state 单写方（SG lifecycle 独占 admission 写）。
- 桩测：乱序 proposal-eval 消息自包含 apply 成功；重复 seq 幂等
  跳；per-kind 游标各自追平互不卡。
- 迁移注册+迁移测试过 `scripts/check_all.sh`；`docs/数据流.md`
  桥面段同 commit 重写。
- **生产锚**：实桥面提案/评估 lag 可见+断线恢复实证（含回滚径
  演练一遍——单命令回 dualwrite 待终切后裁撤）。
