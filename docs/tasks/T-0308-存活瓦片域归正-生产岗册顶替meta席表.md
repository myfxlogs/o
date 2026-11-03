# T-0308 存活瓦片域归正——生产岗册顶替 meta 席表

## 状态

**已批（决策立案——业主指认：存活瓦片值域错）**

## 背景（生产实证 2026-11-03）

业主驾驶舱「① 存活」读 `ops_seat_latest`+`ops_headless`=**meta 域**
（协作席自报+dev 无头会话）——22 席是 agent 协作席计数非生产岗；
「9 异常」红警源=11-01 06:58 批量判死的会话尸体（eval-20261029/
T-0220b-d/T-PROBE 等 dev 会话坟场），非生产岗死亡。生产真名册=
`fin_employee_registry`（23 active+1 retired，30s 心跳）——T-0307
活水段已消费真源，存活瓦片仍读假源=同屏两「存活」打架。

## 钦定目标

1. **存活瓦片改读生产源**：主数=`fin_employee_registry WHERE status='active'`
   count（23）；副行=最坏心跳 lag（同 GetFlowPulse 口径——复用勿双写）；
   红警条件改=**任一 active 岗 last_beat 停滞超阈**（如 >10min——钉常量），
   不再数 meta 尸体。
2. **无头面子行处置**：ops_headless 数移出 KPI 红警径——保留显示则改中性
   灰标「dev 会话 X 在飞」（dead/stalled/overdue 尸体不参警）；或整行撤
   （meta 面属 ops 域面非业主 KPI——二选一，票面钉）。
3. 验收锚：存活瓦片主数=23；杀任一服务（如 docker stop o-collecto）
   >10min 瓦片转红显该岗名（演验证录票面）；ops_headless 尸体不再制红。

## 钉注

- `fin_employee_registry` 行 TTL：retired 行不入数（集使 10-24 退在册——
  WHERE status='active' 已验取值域）。
- 勿并 GetFlowPulse 请求（瓦片是 GetCockpit 首帧件——同函数内查同名
  SQL 式即可，roster lag 口径与活水段一致）。
- ops_seat_latest/ops_headless 不删不改写——meta 面留运维排查用，
  仅出业主 KPI 视线。

## 施工回填（devin-t0308 @ 2026-11-03，同支 t/0307 续提）

- **存活瓦片主数**：`fin_employee_registry WHERE status='active'` count——
  GetCockpit 内查同名 SQL 式（roster_active/worst_lag_s/worst_seat 三字段
  入 proto field 10-12——与 GetFlowPulse 口径同源勿双写；读败=missing
  "roster"）。
- **红警条件**：active 岗 last_beat 停滞>10min（600s 钉常量）→瓦片右上
  转红显该岗名+滞后；lag 文本照显。ops_headless 尸体不参警。
- **无头面子行**：改中性「dev 会话 X 在飞（meta 排查面不参警）」——
  dead/stalled/overdue 尸体出红警径不制红（择中性标项）。
- **顺修**：cockpit_face_test 场景件清基（fin_0097 ops_event 种子行虚增
  tickets_total——同 pipeline 测试隔离件）。
- 测试：core 全绿@5434；tsc --noEmit 绿；杀服务演验=部署后实证件。
