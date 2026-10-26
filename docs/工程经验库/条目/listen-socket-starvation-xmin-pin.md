---
schema_version: 2
id: listen-socket-starvation-xmin-pin
kind: pitfall
risk: high
status: verified
keywords: [LISTEN, 读饥饿, ClientWrite, backend_xmin, 专读 goroutine]
scope: [internal/platform/events, 一切 PG LISTEN/NOTIFY 常驻监听者]
summary: LISTEN 环 fn 同步串行于读位→fn 长拍期间套接字零读→ev_* 通知洪泛塞满接收缓冲→服务端 notify 交付的隐式事务卡 ClientWrite 收不出→通知方 xmin 钉死 vacuum 水平线——与池事务残留（pool-tx-residue-vacuum-pin）同症不同根的姊妹病灶；修=读/worker 解耦（专读 goroutine 持续排干→cap1 唤醒位塌陷，一醒一拍背压不变）
first_seen: 2026-10-26
last_verified: 2026-10-26
evidence: [T-0154, SG pid 124436 现场终勘 wait_event=ClientWrite+xmin 钉, 跨容器复现 pid 120093 同形]
---

# listen-socket-starvation-xmin-pin——LISTEN 读饥饿钉死 xmin（notify 交付 ClientWrite 卡死型）

## 症状
- xmin 哨兵命中 LISTEN 臂（pool-tx-residue 同形告警），但**守卫/自净零 WARN、
  LISTEN 注册干净**——不是池残留，别往池卫生方向修。
- 钉手会话 wait_event=**ClientWrite**（服务端往这个客户端写不出）、
  query_start=xact_start=一次通知交付拍时；同池其余 LISTEN 会话 ClientRead 健康。
- 杀掉钉手后监听臂**零「事件听断」日志**——监听 goroutine 不在
  WaitForNotification/Exec，卡在 fn 内部（conn 死对它无感）。
- 长拍任务（扫描 20min+）的事件臂+高频通知频道（ev_* 洪泛）同现时必发。

## 根因链
1. 事件环 fn 同步串行于读位（WaitForNotification 返回→fn→再等）——fn 一拍
   20min，期间**同一连接套接字零读**。
2. ev_* 通知洪泛塞满服务端→客户端接收缓冲（Recv-Q/TCP 窗）。
3. PG notify 交付=通知方 backend 在**自己事务提交内**直写订阅方套接字；
   写不出→通知方整个提交收不出→**通知方 xmin 钉死** vacuum 水平线
   （热表死元组只堆不收——VACUUM FULL 也无效）。
4. 监听臂自身「假活」：注册正常、goroutine 在跑，只是读不动——
   连接在/goroutine 在等活性判据全过，实害照发。

## 排查指纹
```sql
SELECT pid,state,wait_event,backend_xmin,now()-xact_start,left(query,40)
FROM pg_stat_activity WHERE backend_xmin IS NOT NULL
  AND xact_start<now()-interval '10 min';
```
wait_event=**ClientWrite** 的钉手=本型实锤（且其 query 是普通业务语句而非
LISTEN——是被交付卡住的通知方）；ClientRead/空闲持留形=poll-tx-residue 型
（查还池事务残留）。两型分诊靠 wait_event，治法完全不同。

## 修法（T-0154 返修）
- **读/worker 解耦**：专读 goroutine 独占套接字持续排干 WaitForNotification→
  通知塌陷进 cap1 唤醒位；worker 等唤醒位跑 fn——积压合流一醒，「一醒一拍」
  背压语义不变，fn 时长与套接字排空解绑。
- 配套：自净 ROLLBACK 败→弃连不 proceed（堵吞错洞）；LISTEN 注册毕断言
  TxStatus=='I'（事务内注册永不生效=全聋，拒启）；哨兵报文带
  wait_event/backend_start 取证粒度。

## 教训
- **「同步串行=天然背压」在 LISTEN/NOTIFY 面有隐藏代价**：反压的不是自己的
  事件流，是**服务端交付事务**——慢一拍的代价由全库 vacuum 水平线代付。
- xmin 钉手分诊先看 wait_event：ClientWrite=交付反压型（本条）；其余=
  快照持留型（pool-tx-residue 姊妹条）——同症（xmin 钉+监听异常）不同根。
- 监听臂活性判据再+1：**读位是否被工作挤占**（goroutine 活≠套接字在读）。
- 桩锚形：fn 阻塞 ~30s 期间灌 7KB×15ms 通知——旧码发射 Exec 卡死（交付
  反压实锤）；新码发射零反压+listener 零 xmin 钉+积压塌陷恰 2 拍。
