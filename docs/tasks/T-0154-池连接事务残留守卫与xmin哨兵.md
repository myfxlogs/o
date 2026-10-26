# T-0154 池连接事务残留守卫 + backend_xmin 哨兵（真空域钉死双病灶根修）

## 态：✅ 验收毕投产（35e7f25 合入·决策席 2026-10-26）
## 领取：zcode-t0154（ZCode 施工席）@ 2026-10-26
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

## 结果回填（zcode-t0154 @ 2026-10-26）

- **守卫件 pgguard（新独立包）**：GuardPool(cfg)——池配 PrepareConn
  借出前 TxStatus 检查（BeforeAcquire 已弃用 v5.10，SA1019 归 PrepareConn
  本位）：非 idle→ROLLBACK 清态放行+log 计数（TxResidueCount 观测面——
  抓泄漏借用者定位面）；ROLLBACK 败=销毁连接。独立包形（pgstore/
  testutil/cmd 自建池共用——互引环引隔离实勘后定）。
- **三处挂线**：①pgstore.Connect 默认挂；②cmd/collector 自建池
  （pgxpool.New→ParseConfig+GuardPool+NewWithConfig）；③testutil.Pool
  同守（测试面桩锚同池验）。import 闸 +2 边（pgstore/testutil→pgguard）。
- **listener 双保险自净**：listenOnce/listenOnceDeadline Acquire 后
  TxStatus 检查→ROLLBACK（事务内 LISTEN 永不生效=聋哑双实害源防御）。
- **xmin 哨兵**：pgguard.Sentinel（10min 节拍 sched-exempt-ok 观测拍）——
  pg_stat_activity xact_start>10min 命中→Alert(pool_tx_residue) 一次
  （恢复清标再报抑刷屏）**先报不杀**（杀权留人工——票面钦定）；挂
  cmd/core（业主通道）+nil log 退路。
- **桩锚**（PG 实跑全绿）：TestTxResidueGuard（直协议 BEGIN 带病还池→
  下任借用者 TxStatus='I' 干净——卫生结果断言）；TestXminSentinelQuiet
  （零钉手零警静默语义）；pgstore/testutil/events 回归绿。
- **实勘归正（防再议）**：pgx v5.10 Release 已自毁带事务态连接
  （pgxpool/conn.go Release：TxStatus!='I'→res.Destroy）——canon 病灶
  出自旧版/旁路直协议径；守卫=版本差异防御纵深，桩锚计数诚实化为观测面
  （卫生结果断言为锚——双层任一生效即绿）。canon 条目事实补注随票。
- **机检**：build✓ vet✓ gofmt✓ staticcheck✓（SA1019 归 PrepareConn 本位）
  check-lines✓ import 闸✓（+2 边）D-0008-B✓（哨兵观测拍豁免先例径）。
- **移交面（部署后决策席实证）**：生产零 backend_xmin>10min 残留（哨兵
  静默=对）；热表体积回正常量级不复胀；事务态还池日志计数可观测。

## 哨兵首报分诊（决策席第 2 席 @ 2026-10-26 17:2x CST——移交面「部署后实证」项启动）

- **告警**：pool_tx_residue 09:24:36Z——pid 120093 LISTEN ev_eval 会话事务超 10min。
- **时序判定**：哨兵/守卫/自净同一 commit（35e7f25 15:30 CST），告警=17:24 CST
  （验收后 1.7h）——**哨兵活着=新码在跑；容器模型下旧会话不可能跨 force-recreate
  存活 → 指向新码复发（守卫覆盖漏或自净失效）**，非历史残会话。
- **码面推演**：全仓唯一 ev_eval 监听者=lifecycle ListenLoop（run.go:47）；
  借出 TxStatus 双检+LISTEN autocommit，推演给不出开事务径——**实证压倒推演，
  待 SG 亲证**：①该 pid state/xact_start/app/client_addr ②镜像 ID+CreatedAt
  vs 35e7f25 时点 ③会话年龄 vs 容器年龄（会话<容器年龄=新码复发实锤）。
- **裁决**：①杀权授权——kill pid 120093（该臂钉+聋双重害，ListenLoop 断线
  自愈 boot 首拍重对齐，零数据风险；本席 ssh 被权限界拦两次，执行明提条件
  请求）②若实锤新码复发→本票返修单（守卫覆盖漏根修），哨兵维持先报不杀
  ③「升自动」暂缓至返修验证毕。
- **移交面原项仍挂**：生产零 backend_xmin>10min 残留未达成（首报即违）——
  本票不计完成，状态待返修裁。
