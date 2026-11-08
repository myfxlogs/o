# T-0468 测试族分库消 PG advisory 锁 herd——check_all 全谱 40min 根修

- 状态：待验（施工收口 2026-11-08——移交外审席复审）
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

## 结果回填（施工 2026-11-08）

### 实现摘要（分库在 testutil/lease 层透明注入，check_all.sh 零改动）

- **`internal/platform/testutil`**：`O_TEST_SHARDS=K>1` 时按调用方
  包目录 fnv1a32 确定性散列落 `o_test`（K=1/散列 0）或
  `o_test_s1..sK`；`ensureShardDB` 首调自确保建库（连 postgres
  维护库 `CREATE DATABASE`，进程内互斥+成败同记防重试风暴，
  并发 CREATE 撞码复查放行；确保败**落基库 o_test=只慢不语义
  错**）；库名 `^o_test_s[0-9]+$` 白名单校验。`ShardDatabase()`
  公共面供裸 DSN 第二池钉库。
- **裸 DSN 第二池根修×2**：`cmd/core/pipeline_face_test.go`、
  `internal/venue/onboard/writefail_test.go` 各 +1 行
  `cfg.ConnConfig.Database = testutil.ShardDatabase()`——schema
  在分库不在基库，不钉连基库=断言必败（KB 条目同 commit：
  `test-shard-second-pool-must-pin-database.md`）。
- **`scripts/testdb_lease.sh`**：acquire 导
  `O_TEST_SHARDS="${O_TEST_SHARDS:-16}"`（外部预置尊重不覆写）；
  租约/端口锁/外部 DSN 径语义一字不动。env-allowlist 登记。
- **`scripts/gc_test_schemas.sh`**：扫①②③ 库内枚举扩
  `o_test` 族逐库扫；守卫放宽 `o_test_s<N>`（仍测试族命名域
  fail-closed）；扫④ 分库豁免（持久基建非遗留 junk）+基库单跑
  防实例级清理重复。
- **桩测**：`testdb_shard_test.sh` G1–G6（grep 钉+租约导出+
  裸池配对+GC 面+**真 PG advisory 锁命名空间实证**+端到端分库
  实建）+`testutil/shard_test.go` 5 断言纯函数映射钉。

### 判据① 墙钟对比（同硬件同树同端口背靠背 A/B，`-count=1` 未暖缓存）

五包 PG 重载子集（lifecycle/projector/earn/pgstore/onboard），
同 worktree、同 `O_TEST_PG_DSN`→5434、串行两跑：

| 档 | 墙钟 real | lifecycle | projector | earn | pgstore | onboard |
|---|---|---|---|---|---|---|
| K=1 基线（全挤 o_test） | **5m33.694s** | 300.3s | 333.2s | 332.3s | 90.2s | 126.2s |
| K=8 | 4m29.943s（-19%） | 239.4s | 270.1s | 269.3s | 99.4s | 113.5s |
| **K=16（定稿）** | **2m16.075s（-59%）** | 132.9s | 135.7s | 133.9s | 31.7s | 60.9s |

K 选型数据驱动：go test `-p`≈GOMAXPROCS(12) 并发包上限封顶，
K=16 使并发 PG 包各得锁域（哈希碰撞余量），再高无收益只增
建库面。check_all 集成跑（mode=scoped 绿戳在案，见判据③）
15m51s@K=8 旧码——本 commit 改 K=16 后以新 SHA 复跑取戳。

全谱 40min 病灶窗口不可复刻（fleet 拥挤度时变），判据①以
同树背靠背 A/B 为准——对照更干净；如实报：未取得「全谱
40min→X」直接对拍数，机制级改善幅度 -59%（子集）在案。

### 判据② advisory 锁等待观测下降（实证两面）

- **运行时采样**：K=1 跑期间 `pg_locks` 采样 o_test advisory
  等待会话持续 1–3（16:29–16:31 窗口多拍实证）；K=8 跑期间
  全测试族库等待合计 ≤1（单分片 o_test_s6 尾部残留）。
- **锁命名空间桩证（G5 真 PG）**：同键 860223 跨库
  `o_test_s97`/`o_test_s98` 互不阻塞 31ms（<3s lock_timeout）；
  同库同键互斥不破 2029ms 阻塞至 lock_timeout——advisory 锁
  域含 databaseOid，跨库并发 Migrate 真并行，锁语义无损。

### 判据③ 零语义改变/skip-blind 禁报绿

- diff 零测试断言改动——触测试文件仅两处 +1 行
  `ConnConfig.Database` 钉库（基建接线非断言）；判据钉死语义
  论证：包间零共享态由包级 schema 隔离结构性保证
  （`t_<pkg>_<rand>` 独 schema），任意确定性库划分不拆语义耦；
  cluster 级对象（o_fin_ro 角色）租约实例已存在幂等；
  public 残表本就非契约态。
- 分库跑绿戳：check_all 真跑（租约 DSN 径非桩），
  `O_TEST_SHARDS` 由 lease 导出，DB 测试打真分库
  `o_test_s<N>`（`\l` 实证 s1..sN 实建）——非 skip 假绿；
  G6 端到端桩在 check_all 环境自带真 DSN 下实跑。
- 已知环境噪声如实报：首轮 scoped 跑撞
  `internal/platform/events` `TestListenSocketStarvationRootFix`
  红（通知反压 context canceled——存量 flake 非本 diff 引入，
  后续跑绿）；fleet 并发 check_all 时窗致墙钟读数偏高，
  已用背靠背 A/B 消混杂。

### 判据④ 既有档制/锁机制/lease 语义不破

- `scripts/check_all.sh` **不在 diff**——档制/树锁/绿戳机制
  零改动；分片对 go test 透明（`-p` 由 go test 原生
  GOMAXPROCS 控制，票面「落点候选」标注施工实裁定，本实现
  选 testutil+lease 注入径优于编排分批：零编排面=零新故障域）。
- `testdb_lease`：外部 DSN 径/端口锁域/等待超时全保留；
  `O_TEST_SHARDS` 预置值尊重（G2 桩双断言）。
- `O_TEST_SHARDS<=1` 或分库确保失败→恒基库 o_test=现行语义
  逐字节保留（fail-safe 只慢不语义错）。
- `gc_test_schemas.sh` 守卫 fail-closed 不放宽命名域外；
  扫④ 基库单跑防重复实例级清理。

### 偏离/裁定记录

1. **check_all.sh 并发编排未改**——票面落点候选③「按域分批
   并发」裁读为不需要：分库后 go test 原生 `-p` 即天然按包
   并发且各自落独立锁域，加编排=重复造轮+新故障域。
2. **K 默认 8→16**——初版 8 经 A/B 三档实测后数据驱动上调
   （4m30→2m16），注释钉选型依据。
3. **顺手修** `testdb_lease_test.sh` E 场景桩补
   `check_seat_writes` 桩单——master 存量桩腐（T-0145 闸件后
   未补），基线红非本票引入，范围内顺手根修。

### 绿戳/推送

- 代码 commit（rebase 后）：`8227c61fd`；本回填 commit 见
  支顶——check_all 绿戳对口最终 HEAD SHA（戳件名内嵌）。
- 未 merge 未部署；串行施工遵令。
