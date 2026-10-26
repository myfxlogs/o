# T-0154 池连接事务残留守卫 + backend_xmin 哨兵（真空域钉死双病灶根修）

## 态：待施工
## 域：平台池基建 / 数据面
## 源出：数据体积排查实案（2026-10-26）——dev 库 LISTEN ev_eval 会话
  backend_xmin=2274019 钉 14.5h（当时 xid 已 8.66M），VACUUM FULL 都收不了；
  杀会话后 fin_opp_eval 1332MB→4MB（99.7% 是钉住的死元组）。SG 同病：
  ev_eval 听会话 xmin 钉 13min 被抓当场——**且钉住期间监听是聋的**
  （事务内 LISTEN 要 COMMIT 才生效——不结事务=永不生效+永不醒）。

## 病灶
池连接被上任借用者留了未结事务还池（pgxpool Release 不检 TxStatus），
ListenLoop 借到即继承——此后 WaitForNotification 永不结束事务：
①backend_xmin 钉死→该连接存活期 vacuum 水平线锁死→全库死元组堆积
（dev 3.7GB 里 ~86% 是这病灶+schema 残留）；
②LISTEN 在事务内不生效→该臂对订阅频道全聋（SG 抓到的是开仓臂
 ev_strategy/ev_eval——开仓面真空期）。双重实害。

## 施工
- `pgstore.Connect`：池配 `AfterRelease`（或 `BeforeAcquire`）守卫——
  `conn.PgConn().TxStatus() != 'I'` → 销毁该连接（return false）或
  先 Exec ROLLBACK 再放行；**记日志计残留来源**（log conn 在事务态
  还池事件——抓泄漏借用者靠此计数定位）。
- 同源普查：`cmd/collector`/`testutil` 自建池同守。
- **xmin 哨兵**：巡扫面加一拍——`pg_stat_activity WHERE backend_xmin
  NOT NULL AND xact_start<now()-10min` 命中即告警（先报不杀，杀权
  留人工/后续裁定——LISTEN 臂自杀有重连兜底可后续升自动）。
- 顺带：listener 借到 conn 后可先 `ROLLBACK` 防御一笔（listenOnce
  Acquire 后 TxStatus 检查自净——双保险）。
- 桩锚：借 conn→BEGIN 不结→还池→守卫销毁/Rollback 实证+下任借用者
  TxStatus='I' 干净。

## 验收锚
- 生产零 backend_xmin>10min 残留会话（哨兵静默=对）。
- 热表体积回到行数正常量级且不复胀（fin_opp_eval 量级 MB 非 GB）。
- 事务态还池日志计数可观测（抓真凶借用者后续专票根修）。
