# T-0434 hostmetrics 监控自监——写器死活根修 + 前端陈旧硬闸

- 状态：施工中
- 领取：claude-t0434 @ 2026-11-07（外派施工 O-Dev 径，worktree
  /opt/o-wt/t0434 支 t/0434-hostmetrics-selfprobe——基于
  origin/master 落 7461c873 后并入本地 master 9675dab1 使票面
  commit 成祖，偏离说明：origin/master 未含立票 commit，不并
  无法翻转票面，已核实 9675dab1 为 docs-only）
- 优先级：P1——system 页渲染 8 天前尸体数据（fin_host_metrics
  'sg' 行冻结 2026-10-30 04:18，df 实拍 97% vs 页面假数）；
  观测面撒谎=运维决策基于尸体
- 收口移交归：外审径（业主令 2026-11-07「外派外审外部署，
  不去给DM做积压」成例——外派件闭环=外派施工→外审席裁→外部署）

## 病灶（2026-11-07 实证三查）

- `fin_host_metrics` 唯一行 host='sg' ts=2026-10-30 04:18，
  disk_total≈450G/disk_used≈252G；同期 `df` 实拍 used≈411G——
  前端展示的「磁盘余量」是 8 天前的尸体；
- 写器=`o-core` 内 `hostmetrics.Run(ctx,pool,"sg","/",30s)`
  （cmd/core/main.go:132）——o-core RPC 仍正常应答，
  **只有自报 goroutine 死了**；
- 死因设计：`Run` 里 `pool==nil→return` 一次永退（无重试）、
  `Upsert` 失败仅 log（「观测面缺不拖业务面」钦定静默）——
  **盯人的探头没人盯**，静默死第五条实证；
- 前端：API 已回 `Stale` 标（ts 龄>2min），但页面照常渲染
  旧数值——标在尸体仍展出。

## 面（钦定，三件一体）

① **写器死活根修**：先实锤 10-30 死因（o-core 日志/启动序
   pool 就绪时序），按根因修——禁止只加 log 糊弄：
   - pool 未就绪不永退：带退避的重试环直到首拍成功；
   - Upsert 连败阈值→ops_event 升件（写不进库=观测断供
     是一等事件，非 log 噪音）；
   - 断言红桩：模拟 pool nil 启动→恢复后自报环必活；
② **监控自监探针**：`fin_host_metrics.ts` 陈旧→ops_event
   （复用 feedstale/ctx-probe 同构形——第三实例即成族，
   抽公共「自报表陈旧→升件」模子勿第三份拷贝）；
③ **前端陈旧硬闸**：Stale 行禁正常渲染——整块置灰+红标
   「数据陈旧·非实时」，数值可留但视觉层级必须让人一眼
   识破是尸体；无行=显式「无数据」态不空跑。
   API 侧 Stale 语义不动（已正确），改的是渲染契约。

## 验收

- 10-30 死因实锤结论回填票面（不许「原因不明就修了」）；
- 桩测：pool nil→ready 恢复自报环活；Upsert 注入连败→
  ops_event 行落；fin_host_metrics.ts 手工回拨>阈→升件；
- 前端：stale 行渲染断言（灰+红标存在）+fresh 行零回归；
- 绿戳对口+全链；坑处置随 commit（静默死族新实证若
  够格→坑条目）。

## 备注

「自报件陈旧→升件」三实例成族后，建议顺手在票面留一句
出生律候选：新自报表若无陈旧探针=交付不齐——本票不强制
立律，留给外审席判。

## 结果回填（claude-t0434 @ 2026-11-07）

### 死因实锤结论（不许「原因不明就修了」——已实锤）

**真死因=自报写器宿主进程死亡且无重启，非 goroutine 级死亡**。证据链：

1. 尸体行落点=**dev 机 o-postgres**（非 SG 生产库）：`SELECT host,ts,disk_total_gb
   FROM fin_host_metrics` @dev → 唯一行 'sg' ts=**2026-10-30 04:18:44 UTC**、
   disk 449.6/252.0——与票面三查数值逐位吻合（450G/252G/04:18 唯一行）；
   同表 @SG → 两行（dev 44.07G/sg 57.97G）俱鲜。票面病灶观察面=dev 栈库。
2. 450G 盘值=dev 机盘（450G/407G/96% 现拍亲证）；SG 盘=58G——写者物理在 dev。
3. dev 机现况：无 :50051 监听、无 o-core 进程、无 o-core systemd/tmux 单元——
   写者已死 8 天无人复活。写者本体无痕可考（现存容器遗骸 Exited(0)@10-29
   14:36 UTC 死于冻结前 38h——其首拍若落账行 ts 应为 10-29 14:36 而非
   04:18:44；冻结期写者=已 rm 容器或裸进程两可，不影响「进程死无复活」定论）。
4. 冻结瞬间（10-30 04:05-04:35 UTC）dev/SG 两侧 journal 零 docker 容器事件——
   非重启致死；SG 侧 PG 错误日志全窗（10-30→11-07）对 fin_host_metrics
   **零报错**——非写败致冻，是无人写。
5. 票面「o-core RPC 仍正常应答，只有自报 goroutine 死了」与实况有偏差：
   RPC 应答面=SG 栈（SG o-core 容器 10-31→11-06 多次 rc=0 重建、今天 02:12
   后写器健康——日志零 hostmetrics 失败行、行鲜）；尸体在 dev 栈（进程死全栈
   无写者）。两栈同名同表（o-postgres/o-core/fin_host_metrics 全同名）——
   栈身份不钉即伪证（坑条目 stack-identity-unpinned-false-evidence）。
6. 票面点名的三个设计缺陷全部实证为真（码面在案）：pool==nil→return 一次
   永退、Upsert 败仅 log、本表无陈旧探针——任一命中即同形静默死，照修。

### 交付三件

① 写器根修（internal/platform/hostmetrics/{hostmetrics,probe}.go）：
   `Run(ctx, poolFn func() DB, ...)` 惰性取池——nil 退避重试环（interval 起步
   封顶 10×interval）不永退；Upsert 连败≥3 拍→ops_event 观测盲升件
   （scope=HOSTMETRICS-<HOST>，24h 同窗去重；升件写败=观测臂同瞎如实 log）。
② 自监探针：**selfprobe 公共模子抽取**（internal/platform/selfprobe——
   row-age 巡检+24h 去重+feed断更行+观测盲升件四拍归一），feedstale(T-0410)/
   ctx-probe(T-0429) 改挂模子（回归绿），第三实例=fin_host_metrics 停拍探针
   （阈 10min=30s 拍×20，容忍部署窗重启+健康闸 90s；core/collector 经
   hostmetrics.Start 一体装配**双挂对机互探**——任一写者进程死=对机活者
   代吼，进程死不再逃逸观测）。
③ 前端硬闸（web/src/react/components/hostmetrics.tsx+20-cards.css）：
   stale 行数值区整块置灰（hm-stale-body：opacity .45+grayscale）+红标
   「数据陈旧·非实时」（hm-stale-badge 满强度不随灰衰减）+停拍 hint；
   无行=显式「无数据」态；**客户端 2min 龄复核与服务端标双判**——断流
   僵尸帧（服务端哑火推不了新标）同被闸。API 侧 Stale 语义未动 ✓。

### 四断言红桩对账

- ① pool nil 启动→恢复后自报环必活：**PASS**——TestRunPoolNilThenRecover
  （hostmetrics_test.go：poolFn 前 2 拍 nil 后真池→sg 行落）。
- ② Upsert 注入连败→ops_event 行落：**PASS**——TestRunUpsertFailEscalation
  （failHostDB 注入缝：fin_host_metrics 全败/ops_event 透传→HOSTMETRICS-SG
  升件行落账）。
- ③ fin_host_metrics.ts 手工回拨>阈→升件：**PASS**——
  TestProbeStaleEscalationOnRolledBackTs（回拨 1h>10m→HOSTMETRICS-SG 升件；
  鲜行零报/零行史同报/24h 去重四形全覆盖）。
- ④ 前端 stale 渲染断言（灰+红标存在）+fresh 零回归：**PASS**——
  web/scripts/hostmetrics-domtest.mjs 11 断言（stale 红标/置灰壳/拍断语/
  数值可留/僵尸帧闸/鲜行零回归/无数据态），web npm test 全链绿。
- 机检：check_all 全谱绿戳绑 HEAD（见收口行）+cmd/core 300 行棘轮合规
  （probe.go 拆分）+D-514 白名单三边（同向形）+D-0008-B 三处行内豁免
  （退避/心跳/桩测等待）。

### 偏离说明

1. 支基：origin/master(7461c873) 落支后并入本地 master(9675dab1)——origin
   未含立票 commit 无法翻转票面；9675dab1=docs-only 已核实。
2. store.ts useSyncExternalStore 补第三参 server 快照（同函数——SSR/静态渲
   安全，domtest 径；浏览器行为零变）。
3. 探针双挂（core+collector 对机互探）而非单挂：票面「成族抽模子」字面内，
   且覆盖本票真死因（进程死）形态。
4. 出生律候选（票面备注径）：**新自报机/自报表无停拍探针行=交付不齐**——
   立律归外审席判。

### 条件请求（外审席/业主裁）

- dev 栈 o-core 死进程（10-30 04:18 UTC 起）复活或退役=ops 裁量件；其库内
  尸体行随栈处置。**业主所看页面若为 dev 栈径，需先钉页面→栈→库拓扑**
  （本次考古未能亲证业主页面宿主——alpha-server 已排除，dev 侧无面板服务方）。
- o-core 收 SIGTERM 挂死>10s→docker 每部署强杀（journal 144 次，10-27 以来
  ——坑条目 deploy-sigterm-force-kill）：退场链逐环计时未做，留常票候选。
