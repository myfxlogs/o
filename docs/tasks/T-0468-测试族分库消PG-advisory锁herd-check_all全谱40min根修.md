# T-0468 测试族分库消 PG advisory 锁 herd——check_all 全谱 40min 根修

- 状态：已批（业主令 2026-11-08：「立外派票，外部审，不去拖 CTO」）
- 归口：外派外审径（工程面——测试基建/脚本族，非钱径）
- 领取：外派

## 病灶（11-08 全天实证）

- check_all 全谱段 20–40min，其中 go test 大头非 CPU 而是 **PG advisory
  锁羊群串行**：全谱测试包挤一把 advisory 锁（实证锁 key=860223 在包间
  串行传递，每包持锁数秒、余者 futex_wait），并发跑越多队越长；
  cmd/core 曾报 30min 超时（T-0460 工人异常行在案）。
- T-0466 裁定 herd 面「先监控不互斥」——监控已立项，**跑得慢本身
  无人修**。本票修「快」：分库并行把串行段消掉。
- 影响面不止慢：herd 是 check_all 复跑税+无头席傻等+部署收敛口
  排队的共同放大器（跑步机已随 T-0467 绝种，本票消残余时延面）。

## 方案钦定方向

测试库租约族已按 worktree 分端口（5435–5438 实证在案），**族内分库**
是同一思路的下一层：

1. 按测试包域分库/分锁——同域包内串行可保（共享态语义不破），
   跨域包并行跑，互不抢同一把 advisory 锁；
2. 落点候选（施工实裁定）：`scripts/testdb_lease.sh` 族租约口径
   扩「域→库」映射；`internal/platform/testutil` Pool/DSN 分配面；
   check_all go test 段按域分批并发（`-p` 上限随载）；
3. 判据钉死语义不破：凡共享账本/同表断言的包归同域同锁，
   禁为提速拆散语义耦——**宁可慢不可错**。

## 判据

- 同硬件同树，全谱 check_all go test 段墙钟对比在案：基线 vs
  分库后实测数贴票面（目标量级：40min→≤15min 非硬线，
  实测改善幅度如实回填）；
- advisory 锁观测面（T-0466 gauges 行）实证同刻等待会话数下降；
- 零测试语义改变：全谱跑结果同基线（零新增 FAIL/SKIP——
  DSN 租约径真跑，skip-blind 禁报绿，T-0441 裁记病灶在案）；
- 既有 check_all 档制/锁机制/testdb lease 族语义不破。

## 范围

- `scripts/testdb_lease.sh` 及族租约口径；
- `internal/platform/testutil`（Pool/DSN 分配面）；
- `scripts/check_all.sh` go test 段并发编排；
- 桩测新件（分域映射/锁等待观测断言）；
- 本票回填。

## 禁线

- 禁动测试断言语义/共享态归属判据以提速；
- 禁 merge/禁部署（外派收口移交外审席）；
- 钱径面零涉。
