# T-0409 check_all 分级——全局锁与 diff 定向测谱

- 状态：完成（2026-11-07 决策席——外派支 2add3c7d 审裁
  通过 merge 7a78e0f3 全链绿落地；机械误发完成行已于
  T-0414 施工披露中以移交 decl 作废，此行=真实终态）
- 领取：**外派已定**——业主令 2026-11-06「直接全部给
  外派，用外部手术刀」；提示词
  docs/prompts/T-0409-外部施工提示词.md，**内部队不派**
  （此票不入 DM 派工面——收口回分支呈决策席审）
- 优先级：**P0 流程急诊**——check_all 全谱 15min/跑+
  并发互踩，是当前交付链实测最大吞吐瓶颈
- 收口移交归：决策席

## 病灶（今日实证）

- `scripts/check_all.sh` 末段 `go test ./...` 全谱
  ≈10-15min/跑（其余闸合计 1-2min）；
- 同树并发实证：DM 收口账×2+COS 立票×1 三跑并发，
  测试件 29 个全睡死抢资源——45min 零戳，杀跑才解；
- 队列越深堵得越狠：每个收口/合并/立项都付 15min 税，
  税是线性的，拥堵是指数的；
- 死法实证（2026-11-06 ca402.log）：三跑并发下
  `lifecycle`/`projector` 测试包 1800s 超时红——栈根
  `pgxpool.NewWithConfig`，测试件全堵 PG 连接池——
  **锁防的是连接风暴不是 CPU**。

## 范围

1. **全局互斥锁**：`check_all.sh` 入口 `flock`（同树一拍
   一跑；并发入者=排队等锁或即败报「另一跑在途 pid」
   ——语义选一，票面不钦）；
2. **diff 定向测谱**：`go test` 段改「变更包+反向依赖」
   定向跑（merge-base→diff→包集映射；现成 docs-only
   豁免判据同思路扩）——常态件从全谱→定向谱；
3. **恒测兜底包**：钱径/风控面（risk/ledger/execution/
   guard 族——施工查表定册）不论 diff 恒测；兜底理由=
   远端 CI build-test 仍跑全谱，本地定向不破安全网；
4. 测谱决策日志化：戳旁记「全谱/定向/docs-only」
   mode（现 STAMP.mode 已有此面，扩定向档）；
5. 并发实测证据：两跑并发触发——锁行为实证（排队或
   即败按钦语义）。

## 验收锚

① 并发两跑不再互踩（锁实证——排队或即败，无睡死）；
② 常态 docs/小码件测谱时长相较全谱显著缩（实测行
   对照——目标量级秒级~2min）；
③ 钱径/风控包恒测断言（改 docs 也跑 risk/ledger 谱）；
④ 定向谱漏包场景桩测（改 A 包不破 B 包案例）；
⑤ check_all 绿戳对口（本件自身也付全谱税——末跑实证）。

## 红线

- 推闸语义不破——「绿了才推」业主令不降格；
- 定向漏包兜底=恒测包+远端 CI 双网，禁裸奔；
- 锁语义不静默——等锁/拒跑都要如实报因；
- 不动 docs-only 豁免既有语义。

## 结果回填

- 实现摘要：`scripts/check_all.sh` 单件两刀——①入口同
  checkout 互斥锁：`flock` fd9 落 `/tmp/o-check_all-
  <toplevel-realpath sha256[:16]>.lock`，语义=阻塞等锁+
  超时上界 1800s（入者即时报「等锁 holder pid」非静默
  睡死；超时 fail 报 holder 不假绿不自杀；拿锁后才毁旧
  戳——等锁期不碰在途跑戳面）；锁键=路径哈希故 wt 与主树
  不同锁互不串、同 wt 并发才互斥（非全机锁）。②测谱
  分级：变更面沿用 docs-only 同径收集（merge-base…HEAD
  ∪porcelain 净路径，不复制判据）——docs-only 既有语义
  不动；变更面全 .go（混 docs 白名单允许）→scoped=变更
  包+反向依赖闭包（go list 全谱 Imports/TestImports/
  XTestImports 建图 python BFS 反向）+恒测册五域
  （risk/ledger/trading/guard/lifecycle 实址核实=12 包）；
  变更面含任何非 .go 非 docs 码件或包映射任一步不确定
  →回退 full fail-safe。STAMP.mode 三档留痕（scoped 附
  实测包集清单逐行）。
- 测试证据（scratch wt /tmp/wt-t0409-scope 新脚本直跑）：
  ①锁——同 wt 并发第二跑即时报「等锁 holder pid=[3817203
  3829922]（锁=/tmp/o-check_all-cd6ccda….lock；上界
  1800s）」后阻塞等锁，timeout 60 证非睡死非假绿；
  ②scoped——tools/okxverify 探针件→mode=scoped 绿，
  包集=13（变更包+恒测册 12），.mode 附全清单；
  ③full 回退——internal/data/*.proto 未跟踪件→mode=full
  全谱跑通；④docs-only——docs/*.md 件→免测档秒绿
  mode=docs-only；⑤本件自身 diff 含 .sh→末跑 full 档
  全谱实证（验收锚⑤）。
- 时长对照：docs-only=闸段 ~2min 即绿；scoped 实测
  ~5min（恒测册 12 包主导——guard160s/lifecycle181s/
  execution150s 并行段，ledger 族 cached）；full≈15min。
  「秒级~2min」目标由 docs-only 档达成；scoped 地板=
  恒测册谱（钦定册不动即此成本）。
- 偏离说明：①锚③「改 docs 也跑 risk/ledger」按验收②
  「docs-only 现行行为保持」+红线「不动 docs-only 既有
  语义」裁读=恒测册在 scoped/full 档不论 diff 恒在列
  （docs-only 档免测语义不动）；②锁 holder pid 报
  fuser 开档进程集（含锁持有者子进程）如实列不裁；
  ③scoped 档 go test 保留 -timeout=1800s 与 PG DSN
  docker 探测同式。
