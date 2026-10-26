---
schema_version: 2
id: pool-tx-residue-vacuum-pin
kind: pitfall
risk: high
status: verified
keywords: [pgxpool, TxStatus, backend_xmin, vacuum 水平线, LISTEN 聋哑, 池连接污染, 死元组, VACUUM FULL 无效, fin_opp_eval 膨胀]
scope: [internal/store/pgstore, internal/platform/events, 一切 pgxpool 借用者]
summary: 池连接带未结事务还池→ListenLoop 借用继承→backend_xmin 钉死 vacuum 水平线（死元组只堆不收，VACUUM FULL 也无效）+事务内 LISTEN 永不生效致监听臂全聋——表体积假「无底洞」真凶；守卫=Release/Acquire 时 TxStatus 检查销毁污染连接+xmin 年龄哨兵
first_seen: 2026-10-26
last_verified: 2026-10-26
evidence: [T-0154, dev库 3.7GB→537MB 实证, SG fin_opp_eval 721MB→5.4MB]
---

# pool-tx-residue-vacuum-pin——池连接事务残留钉死 vacuum 水平线+监听聋哑

## 症状
- 表体积涨成「无底洞」形：行数千级体积 GB 级（fin_opp_eval 9.7k 行/1.3GB）。
- `n_dead_tup` 不高、autovacuum 狂跑（4000+ 次）也收不掉；**VACUUM FULL 执行成功但体积不减**——这是「有 xmin 被钉」的指纹性信号（区别于单纯 churn 膨胀）。
- 伴随监听臂静默（事务内 LISTEN 不生效→该臂对订阅频道全聋）。

## 根因链
1. 某借用者把**开着事务的连接**还给 pgxpool（Begin 忘 Rollback/Commit——
   Release 不检查 TxStatus，带病连接正常回池）。
2. `events.ListenLoop` 借到该连接→`LISTEN`+`WaitForNotification` 常驻——
   此后无任何语句结束这个继承来的事务。
3. 后果双发：
   - **vacuum 水平线钉死**：backend_xmin=继承事务快照→该会话活着一天，
     全库一天内死元组不可收（autovacuum 跑只是维持不再恶化）。
   - **LISTEN 聋哑**：事务内 LISTEN 须 COMMIT 生效——不结事务=注册
     永不生效=该臂对全部订阅频道收不到任何通知（WaitForNotification
     照样阻塞，表现和「无事件」一模一样——比死循环监听更隐蔽）。

## 排查指纹
```sql
SELECT pid,state,backend_xmin,now()-xact_start,left(query,40)
FROM pg_stat_activity WHERE backend_xmin IS NOT NULL
  AND xact_start<now()-interval '10 min';
```
命中 LISTEN/长驻会话=xmin 钉实证。`pg_terminate_backend(pid)` 后
VACUUM FULL 即回收（LISTEN 臂有断线重连兜底，杀安全）。

## 实勘补注（T-0154 施工 2026-10-26）
pgx v5.10 Release 已自毁带事务态连接（pgxpool/conn.go：TxStatus!='I'→res.Destroy）——本条病灶出自旧版/旁路直协议径（PgConn 层 BEGIN pgx 不可见形）。守卫（pgguard.GuardPool PrepareConn）=版本差异防御纵深+观测计数面，卫生断言锚在下任借用者 idle。

## 修法（T-0154）
- 池配守卫：`AfterRelease`/`BeforeAcquire` 检 `TxStatus()!='I'`→销毁
  连接或 ROLLBACK 放行（并 log 计数定位真凶借用者）。
- xmin 年龄哨兵：上述查询巡扫化，命中告警。
- listener 借连接后 TxStatus 自净检查（双保险）。

## 教训
- **「表体积无底洞」先查 xmin 钉再查 churn**——两者治法完全不同：
  churn 平台期接受/调 avac，xmin 钉须找钉手（VACUUM 解决不了）。
- 池连接污染不只传频道（T-0131 UNLISTEN 先例），**事务态同样继承**——
  一切会话级状态都是污染面：LISTEN 注册、事务快照、search_path、
  prepared 冲突。守卫要对症全谱。
- LISTEN 臂活性判据再+1：xmin 年龄（注册正常但事务不结=假活）。
