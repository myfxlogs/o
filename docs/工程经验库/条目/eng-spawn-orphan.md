---
schema_version: 2
id: eng-spawn-orphan
kind: pitfall
risk: medium
status: verified
keywords: [spawn_orphan, 唤知, 唤起孤儿, release_lock, trap, "30min TTL", "|| true", 死场残渣, RuntimeMaxSec, 会话收口]
scope: [唤知环生命周期账, cmd/core/audit_esc.go spawnOrphanEscCheck, scripts/cognition/run.sh]
summary: spawn_orphan 升件=唤起行 3h–24h 窗内无配对果/败/收口的死场残渣探——会话吊死或宿主亡,trap 写不了行,残余黑洞由本探升件接管。处置档=①唤起行取 unit 核死(journalctl/sessions.db)②unit 死透+锁已 TTL 自释→销(探针设计内残余非病灶);同因连败→接 cognition-spawn-env-drift-lockout 档;unit 仍活→双会话险,杀后查因。run.sh release_lock 的 INSERT 用 || true 是钦定设计——写败无二级上报径,锁由 30min TTL 兜底,不阻主径退出。
first_seen: 2026-11-18
last_verified: 2026-11-18
skills: []
evidence_refs:
  - type: server_fact
    source: 升件#23854 + o-self-1794960861 unit 账
    detail: models 自席 unit 00:14Z 唤起,devin agent retryable connection error 00:17Z 进程死,EXIT trap 的释锁 INSERT 静默败(|| true),唤起行成孤儿;spawn_orphan 探针 3h 后点火,锁已 TTL 自释,处置=销
---

# spawn_orphan——死场残渣探的处置档

## 探针机理（cmd/core/audit_esc.go spawnOrphanEscCheck）

唤起行落账后 3h–24h 窗内**无配对**（唤起果/唤起败/会话收口 三行
任一载同 unit token）→ 升件。三栏判据：

- **宽限 3h**：RuntimeMaxSec 2h 上限+写行滞后余量——3h 外零合法
  在飞（48h 实测最长合法会话 108min）；
- **24h 回看界**：唤起行出窗孤儿自然转绿销案——残余有自洁期；
- **自锚**：只判「unit 收口契约活后出生」的唤起（锚=首条带
  `unit=` 的会话收口行 ts；契约缺席时判集恒空=探针自闭，老格式
  唤起配不上对不冤判）。

二元死态 fp="@"——销后同回合不火，再孤=新回合照火。

## 为什么孤儿是设计内产物而非病灶

`run.sh` 的 EXIT trap 释锁 INSERT 用 `|| true`（line ~51）——
**写败无二级上报径**：果败写径本身就是败迹时，再为它上报会
无穷递归。锁由唤起行 30min TTL 兜底，不阻主径退出。SIGKILL/
宿主亡/连接死杀进程时 trap 可能跑不完→唤起行成孤儿→
**spawn_orphan 探针就是这个黑洞的升件径**（认知环自死不知死，
自观帧卡旗同源）。孤儿≠新病=旧死被看见。

## 处置三态（升件#23854 实案）

1. **核死**：唤起行 note 取 unit → `journalctl -u o-cognition-<id>`
   ∨ sessions.db message_nodes 断流时刻——判死因而非只看缺行；
2. **死透+锁自释→销**（常径）：残余=探针+TTL 设计内接管，
   销词载死因（connection error/quota/host death）；
3. **unit 仍活**（孤儿误判的反面——真双会话险）：杀进程+查
   trap 为何没落行→要码修；
4. **同因连败**→接 cognition-spawn-env-drift-lockout 档（env 漂/
   锁饿死四坑全案），本档不替。

## 证据

- 升件#23854（2026-11-18）：models 自席 unit o-self-1794960861
  devin agent retryable connection error→进程 00:17Z 死，释锁
  INSERT 静默败，唤起行孤儿 3h 后点火；锁 30min TTL 自释，
  探针+容忍洞工作如设计→销，无码面缺口立票。

## 反证条件

- release_lock 换二级上报径（`|| true` 撤）→孤儿率应降，
  探针判据不变但实案基数变；
- 唤起行 TTL/配对语义重构→探针三栏判据须随同迁。
