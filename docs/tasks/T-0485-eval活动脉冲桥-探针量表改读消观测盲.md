# T-0485 eval 活动脉冲桥——per-template 新鲜度派生表镜像消观测盲（T-0479 裁量径②）

- 状态：已批（CTO 裁量批 2026-11-08——T-0479 呈裁三径取②；径①以
  rejected-only 族 eval 死永盲换消音=劣、径③养告警疲劳=劣，皆弃）
- 领取：无头施工席常径（swe-2-high——projector+migration 面）
- 优先级：P2（族沉默探针日频误报断源——#3226/#2450/#3446/#3445/
  #2869/#2868 六案同签名）
- 收口移交归：决策席
- 立项：T-0479 诊断定案——SG fin_opp_eval 镜像谓词=仅被
  fin_proposal 引用行（D-0030 钦定承重墙，PG-md ~93upd/s 体量闸
  不可全量镜像）→rejected-only 族 SG 侧结构性恒零→族沉默探针+
  eval_zero 量表日频误报；病灶=观测域错位非评估链 bug

## 钦定语义

- 新增小表（如 `md_eval_pulse`）：per-template max(ts)/count(24h)
  派生行——量级=模板数（~10 行），镜像体量可忽略；
- 数据源：projector serve 派生（PG-md fin_opp_eval 侧聚合写——
  走 projector 写域现径，不新开写权）；
- 消费面改读：hunger 族沉默探针（silence.go eval 存活判据）+
  gauges eval_zero 量表——eval 新鲜度改读脉冲表（fin_opp_eval
  镜像行数不再作 eval 活性判据）；
- 「eval 因谓词不可见」与「eval 链死」从此可分：rejected-only
  族脉冲行新鲜=活，脉冲行陈旧=真死——探针语义保真。

## 范围

1. `migrations/` 新表（列集: template/last_eval_ts/eval_n_24h/
   refreshed_at——命名随链序钦定）；
2. `internal/projector/`（eval 拍尾聚合写脉冲行——serve 派生
   现径，谓词承重墙零触）；
3. `internal/portfolio/hunger/silence.go` + gauges 量表读面改读
   （eval 活性判据源切换——探针阈/签名面/类册不动）；
4. 桥册/数据流.md canon 行同 commit 补；
5. 桩测：rejected-only 族脉冲行新鲜→探针静默 断言（修前红
   =SG 恒零误报谱形 fixture+修后绿）。

## 验收

- 桩测断言绿+check_all 对口（migration+projector 面全谱档）；
- 部署后探针同窗不再报 `观测盲:xvenue-carry` 同签名（自验锚）；
- rejected-only 族 eval 死探测能力保留（脉冲行陈旧→真死可报）。

## 红线

- fin_opp_eval 镜像谓词（D-0030 承重墙）零触；
- 探针阈/签名面/类册禁改；
- 不动真径执行/钱径码位；
- 先红桩律适用。

## 结果回填

- 状态：待验
- 施工回填（2026-11-08 无头施工席 @ wt /opt/o-wt/t0485 枝 t/0485-eval-pulse）：
  - 范围①`migrations/fin_0148_eval_pulse.sql`：md_eval_pulse 建表
    （template PK/last_eval_ts/eval_n_24h/refreshed_at——票面列集钦定）+
    projector_rw SIUD、o_fin_ro SELECT 条件授权（fin_0099/0100 同径）；
    domain.tsv 以 bridge 入册（fin 链独表——md 侧无写者/读者不落正表，
    断言⑥ 同形硬清单不在列）；migrate_table 登记 {147}。
  - 范围②`internal/projector/pulse.go` 新件（行数闸拆分——ops.go 撞
    300 棘轮）：qEvalPulse md 正本 per-template 聚合（谓词城外全量面——
    承重墙零触）+uEvalPulse 同键 upsert+refreshEvalPulse；挂点=
    projectEvals 拍尾（serve/dualwrite 双模同径）+Reconcile 顺带
    （eval 链死期 ev_eval 无帧，10min 活性对账兜底 eval_n_24h 不漂）；
    New() want 册补 md_eval_pulse（搬运面 ⊆ 桥册 fail-closed 对拍）。
  - 范围③silence.go：ProbeFamilies 枚举改 pulseTemplates（md_eval_pulse
    全集——全拒族入探针域化石失明根修）；probeFamily 活性判据改
    `last_eval_ts>窗`（镜像行数不再作判据）；evalTemplates 原件保留给
    depenum（depblind 计数面同读镜像=两读同源内洽，票域外残面见下）。
    gauges.tsv 两 eval_zero 行+flow_pulse eval 段灯改读脉冲表
    （探针阈/签名面/类册/量表名零动——仅 query 列换源）。
  - 范围④domain.tsv 册行+数据流.md 桥面 canon 行同 commit 落。
  - 范围⑤桩测：silence_test seedPulseAt 脉冲播种（SG fin_opp_eval 恒零
    谱形即桩底）；形②加镜像恒零断言=rejected-only 修前红谱形+修后绿；
    新 TestProbeFamiliesLesionOnPulseStale=真死面保留断言（验收锚③）；
    projector_test 新 TestEvalPulseBridge=md 全量聚合→SG 脉冲行
    （镜像恒零下脉冲新鲜）+eval_n_24h 窗口径断言。
  - 自测绿谱：check_migrations 七断言过；check-lines 过（pulse.go 拆分后）；
    go build+vet 过；O_TEST_PG_DSN 实库测——projector 包全绿（含新桩）、
    hunger 包全绿（92s 全谱）、cmd/core flow_pulse 两测绿、pgstore 绿。
  - 已知残面（票域外如实记档，归 CTO 裁量另票）：depblind depCounts
    consN/depenum evalTemplates/facesnap DISTINCT 三面同谱读镜像行数——
    枚举+计数同源自洽不新增误报（改单边才会引入新假阳），全拒族对
    depblind 仍不可见=欠报向残面；depblind_test seedEvalAt 收编回本件。
  - 部署滞后预期（如实报非缺陷）：check_bridge_registry ①授权列实机
    审计——fin_0147 未上 SG 前报「SG projector_rw.SELECT on
    md_eval_pulse 缺」，迁移应用+授权落即自绿（T-0214 同型部署序在案）。

## 返修记档（2026-11-08）

- 返修1——迁移号让位 fin_0148：GW-001 段B `fin_0147_ops_ticket_phase`
  先占 147 号（master 在案），本票迁移改名 fin_0148_eval_pulse.sql+
  登记 {148}+文件注释号同步；DDL/授权/探针/量表语义零改。留案同步：
  SG 已由 T-0488 按同 DDL 直放（md_eval_pulse 表+projector_rw SIUD+
  o_fin_ro SELECT 在案，schema_migrations v147 撞号留案决策席）——
  改名后 fin_0148 落库时 CREATE IF NOT EXISTS+条件 GRANT 幂等零副作用。
- 返修1 验证（枝 rebase 落 f7fd69559 上，merge 文本撞点已消）：
  check_migrations 七断言过（链唯一单调 {147,ops}+{148,eval}）；
  check-lines/build/vet 过；check_bridge_registry 四列全绿
  （授权列 26 桥表×projector_rw.SELECT 齐——md_eval_pulse 授权
  T-0488 在案即验锚绿）；
  check_all 全谱单跑至 go test 档两件红皆证非本支：
  ①cmd/core trx_test 卡 Migrate tx（pgx 帧 herd 争用——check_all
  自附 env-contention 标，同件单跑 5s 绿）；②events
  TestListenSocketStarvationRootFix 负载敏 flaky（本支同库
  两跑一绿一红、master 单跑一绿——本包 diff 零触）；
  本支触面包（projector/hunger/pgstore/flow_pulse）全绿在案。
