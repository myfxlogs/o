# T-0154 池连接事务残留守卫 + backend_xmin 哨兵（真空域钉死双病灶根修）

## 态：待验·返修（读饥饿根修六判据——zcode-t0154r @ 2026-10-26，分支 t/0154-listen-read-starvation 已推送）
## 领取：zcode-t0154（ZCode 施工席）@ 2026-10-26；返修 zcode-t0154r（ZCode 施工席）@ 2026-10-26
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

## 首报根修分诊（决策席 @ 2026-10-26 18:2x CST——实证压倒推演，机制改判）

- **现场终勘（pid 124436 全字段）**：state=`idle`、wait_event=`ClientWrite`、
  backend_xmin 钉、`query_start=xact_start=LISTEN ev_eval` 拍时；同池其余
  LISTEN 会话全 `ClientRead` 健康。**杀权执行（第 2 席授权）**：terminate 后
  **90s 零「事件听断」日志 = goroutine 不在 WaitForNotification/Exec——
  卡 fn(r.scan) 内部**（conn 死对它无感）。
- **机制实锤（非守卫漏、非残留借出）**：LISTEN 注册干净（零自净 WARN 实证）；
  fn 同步串行于读环→**扫拍 ~20min+ 期间套接字零读**→ev_* 通知洪泛塞满
  Recv-Q→服务端 notify 交付隐式事务卡 ClientWrite 收尾不出→xmin 钉死。
  跨容器复现（上轮 pid 120093 同形）=**读饥饿型死锁，与池卫生无关**。
- **返修令（判据）**：①读/worker 解耦——专读 goroutine 持续排干
  WaitForNotification→合流唤醒位（cap1 塌陷，「一醒一拍」语义不变），
  fn 时长不再饿死套接字 ②自净 ROLLBACK 败→Release+err 不再 proceed
  （吞错洞实证）③LISTEN 注册毕断言 TxStatus=='I'④哨兵报文加
  wait_event/backend_start（取证粒度）⑤KB 条目记读饥饿钉死型
  （pool-tx-residue 姊妹病灶）⑥scan 20min/拍的 DB 竞争侧案另案观察。
- **移交面仍挂**：零钉未达成——本票置「返修待派」，哨兵维持先报不杀。

## 返修结果回填（zcode-t0154r @ 2026-10-26——返修令六判据逐条）

- **判据①读/worker 解耦**（events.go 重写）：listenSession 件（conn+wake
  cap1 合流位+readErr 回传+cancel/reap 收摊）——专读 goroutine 独占套接字
  持续 WaitForNotification（fn 长拍不歇），通知塌陷进 wake（积压 N 件合流
  一醒），worker 取拍跑 fn——「一醒一拍」背压语义不变；listenOnce/
  listenOnceDeadline 双环同装。还池前必经 teardown（cancel→等读环退出→
  UNLISTEN→Release——读与还池互斥）。期界由 worker 侧 timer 承接（专读环
  全程不中断）。
- **判据②自净堵错**：startListener 借出 TxStatus!='I'→ROLLBACK；ROLLBACK
  败→UNLISTEN+Release（pgx Release 层自毁带病连接）+wrapped err 走重连梯
  ——不 proceed（首修吞错洞堵死）。
- **判据③注册断言**：LISTEN 注册毕 TxStatus!='I'→弃连+err 拒启（事务内
  注册聋哑形拦死）。两环共用 startListener 单点。
- **判据④哨兵取证粒度**（pgguard.Check）：查询+报文补 wait_event
  （ClientWrite=读饥饿型实锤指纹）+backend_start（会话年龄——跨容器复发
  判定锚）逐会话入报；alert kind 仍 pool_tx_residue。
- **判据⑤KB**：新条目 listen-socket-starvation-xmin-pin（姊妹病灶型——
  wait_event 分诊指纹/根因链/桩锚形）+pool-tx-residue 条目补姊妹互引；
  kbgen gen/--check 绿、skills --check 绿。
- **判据⑥**（scan 20min/拍 DB 竞争侧案）：决策席另案观察——本席未动 fn 面。
- **桩锚**（PG 实跑双向实证）：TestListenSocketStarvationRootFix——fn 阻塞
  ~30s 期间 7KB×15ms 灌通知（量>500）：发射零反压（零超时、最大单发延时
  <5s 断言）+listener 会话全程 backend_xmin 零钉/state=idle/末采样
  wait_event=ClientRead（排空指纹）+放行后积压塌陷恰 2 拍（一醒一拍）+
  cancel 收摊不悬挂+还池 TxStatus 干净。**对抗性双向实证：旧码（origin/
  master events.go）跑同测=120s 红悬（交付反卡实锤形）——测试真杀病灶非恒绿**。
- **机检**：build✓ vet✓ gofmt✓ staticcheck✓ check-lines✓ import 闸✓
  （104/111 边零新增——events_test 测试 import 不入闸）kbgen gen/--check✓
  skills --check✓ 全量 go test（三 DSN 本地实例 5434）✓ platform 面
  -race✓。本地测试 DSN=postgres://o:o@localhost:5434（o-postgres-test 容器
  约定；CI 同构 postgres:test）。
- **勘察移交（语义面发现，非本返修改动）**：期界环 T-0048 注释称「fn 每
  事件/到期各一拍」，实码期界到=return nil 弃连重连、期界拍从不跑 fn
  （overdue 时 nextDue=0→livenessMax 再等 10min）——本返修按令「语义不动」
  保留实码、注释改记实况；「判决到期无事件时迟 ~10min+退避才被拾起」是否
  合意请决策席另裁。
- **移交面（部署后决策席实证）**：生产 ev_* 臂零 xmin 钉>10min（哨兵静默
  =对）；哨兵首报报文含 wait_event/backend_start；lifecycle 长扫拍期间 SG
  无 ClientWrite 钉手复发。
