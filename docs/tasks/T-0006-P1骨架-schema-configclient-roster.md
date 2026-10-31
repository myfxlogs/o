# T-0006 P1 骨架——schema 底表 + config client + 花名册 + cmd/core + 六域空壳

| 字段 | 值 |
|---|---|
| 状态 | 审计通过 |
| 提出 | `[角色:决策]` 2026-10-23 |
| 类别 | 系统骨架（P1） |
| 领取 | devin-t0002 @ 2026-10-23 |
| 依赖 | T-0002 门禁先行（机检在写码前装好）；D-0005 env 契约 |

## 背景

P1 目标：公司架构从纸面到可启动——运行时能连库、读配置、自注册上岗。关键路径：config client 是 Day-1 依赖（D-0005 后一切配置走库），花名册是员工机制的地基。

## 涉及面

`go.mod`·`proto/`·`migrations/`·`internal/platform/`·`cmd/core/`·六域空壳 `internal/{data,research,portfolio,risk,trading,ledger,review}/`

## 验收标准

- [ ] `migrations/`：底表 `app_config`/`app_secrets`（代次列+`pg_notify` 触发器）/`fin_employee_registry`——schema 即契约
- [ ] `internal/platform/config`：config client——`LISTEN config_changed`+代次列+断线快照重载+读错 fail-closed；MasterKey 从密钥文件读（路径约定入档）
- [ ] `cmd/core`：boot=`DB_DSN` 连库→config client→`fin_employee_registry` **Register+Beat**（运行时本体=1 号员工）；健康检查端点
- [ ] 六域空壳包就位，import 红线可跑（风控独立/账本独立/研究独立三方向门禁过）
- [ ] proto 首个 service+`buf generate` 链路通（connect-go v2.0.1）
- [ ] 机检全绿（T-0002 闸集）；`go test` 走独立测试实例 5434

## 范围外

- 任何 venue 连接/真实业务逻辑（P2+）；web 前端（P4）；员工 28 岗分批入职（后续单）
- 设计验证：schema/config 面属架构级——施工前须决策席过设计验证（自审 A–I）

## 结果回填

分支 `t/0006-p1-skeleton`（commit e7255d3，已推 origin）。施工面=worktree `/opt/phase-wt/t0006`。

**验收锚对账**：

- migrations 底表 = **PASS**：`migrations/0001_app_config.sql`（app_config+app_secrets 代次列 generation+`bump_config_generation`/`notify_config_changed` 触发器 `config_changed`）+ `0002_fin_employee_registry.sql`；`migrations/migrations.go` embed FS；`internal/store/pgstore/{pool,migrate,migrate_table}.go` 序进迁移器（schema_migrations 基线+advisory lock 串行化——多测试包并发迁移撞 pg_type 唯一索引实证后修）；测试实拍四表落 `phase_test` 实例。
- `internal/platform/config` = **PASS**：`client.go`（Load 快照/Watch=LISTEN+断线重连重载/Get fail-closed 未就绪+缺键/Secret 按次直读密文行）；`masterkey.go`（`PHASE_MASTER_KEY_FILE` 引导键→缺省 `/run/secrets/phase_master_key`，缺件 fail-closed）；测试实拍：未就绪 Get 拒、写库 UPDATE→通知→快照重载 v2 生效（0.45s 内）。
- `cmd/core` = **PASS**：boot 序 DB_DSN→Connect→Migrate→config Load/Watch→`roster.Register("phase-core","platform")`+BeatLoop(30s)→`:50051`。实拍：`curl :50051/`→`ok`（compose healthcheck 同径）；`POST /phase.dashboard.v1.CoreService/Ping`→`{"status":"ok"}`；`fin_employee_registry` 行 active+last_beat 实拍落库。
- 六域空壳+import 红线 = **PASS**：`internal/{data,research,portfolio,risk,trading,ledger,review}/doc.go` 就位；`check_import_direction.sh` 实拍边集冻结——唯一边 `internal/gen/dashboardv1/dashboardv1connect -> internal/gen/dashboardv1`（codegen 固有）入白名单，三红线判据保留，输出 `OK: import direction gate（1/1 边）`。
- proto 首 service+buf generate = **PASS**：`proto/dashboardv1/core.proto`（package `phase.dashboard.v1`，service CoreService/Ping，connect-go v2.0.1 钉版生成——`connect.NewServer`+`connecthttp.Mount` 新 API 实拍）；产物 `internal/gen/`+`web/src/gen/`（gitignored）；`web/package.json` 最小化仅载 protoc-gen-es 2.16.0（buf 链路依赖，build 脚本占位——前端全新设计 P4 票）。
- 机检全绿+测试走 5434 独立实例 = **PASS（带环境说明）**：`go build ./... && go vet ./...` 净；`gofmt -l .` 零出；`check-lines`/`check-env-reads`/`check_migration_contract`/`import_direction` 全过；`go test ./...` 5 pass（DSN 注 `localhost:15434`——本机 5434 被 nox-postgres-test 占用，T-0002 已记；testutil 缺 DSN=skip 语义）。pre-commit 全闸实测（本次 commit 过闸）。

**实测修正（回填留痕）**：① 迁移并发竞态 pg_type 唯一索引撞→Migrate 加 `pg_advisory_lock(860223)`；② connect-go v2 与 v1 API 不兼容（Request/Response 泛型壳取消→裸 message 出入参）——按生成件实拍改 service.go；③ proto 须带 `option go_package`。

审计席：headless @ 2026-10-31

**审计注记（逐锚对账，收口 commit e7255d39 全量 diff 亲核）**：

- 锚①migrations 底表 = **过**：`0001_app_config.sql` 实存——`app_config`/`app_secrets` 双表各带 `generation` 代次列（共享 `config_generation_seq`+`bump_config_generation` BEFORE 触发器）+`notify_config_changed` AFTER INSERT/UPDATE/DELETE 触发器发 `pg_notify('config_changed')`；`0002_fin_employee_registry.sql` 实存；`migrations.go` embed FS+`pgstore.Migrate` 序进登记制（schema_migrations 基线+`pg_advisory_lock(860223)` 串行化+版本幂等跳过），`migrate_table.go` 登记表 {1,2} 无双登。
- 锚②config client = **过**：`client.go` Load 全表快照+代次=max(generation)、Watch=`LISTEN config_changed` 断线回环先 Load 重载兜底（听上窗漏通知补偿）/Get 未就绪·缺键 fail-closed/Secret 按次直读 `app_secrets` 密文（密文不进快照内存面）；`masterkey.go` `PHASE_MASTER_KEY_FILE`（env-allowlist 在册仅指路径）→缺省 `/run/secrets/phase_master_key`，缺件/空件 fail-closed——路径约定注释+白名单双入档。
- 锚③cmd/core = **过**：`main.go` boot 序 `DB_DSN`→Connect→Migrate→config Load+Watch→`roster.Register("phase-core","platform","运行时本体（1 号员工）")`+`BeatLoop(30s)`→`:50051`，序与票面一致；`/` 健康端点回 `ok`；`service.go` connect v2 `NewServer`+`connecthttp.Mount` 挂 CoreService。roster Register=ON CONFLICT 幂等+Beat 未注册行 fail-closed。
- 锚④六域壳+import 红线 = **过**：六域 `doc.go` 齐（`internal/review` 壳系票面涉及面自列件）；`check_import_direction.sh` 白名单冻结唯一 codegen 边 `dashboardv1connect -> dashboardv1`+三红线判据（risk↛trading/research/portfolio、ledger↛trading、research↛trading）脚本内实存可跑，空壳期零跨域边。
- 锚⑤proto+buf = **过**：`proto/dashboardv1/core.proto`（`phase.dashboard.v1` CoreService/Ping+`option go_package`）；`buf.gen.yaml` dashboardv1→`internal/gen`+`web/src/gen` 双产物 gitignored；`go.mod` `connectrpc.com/connect/v2 v2.0.1`、`protobuf v1.36.12` 钉版，`web/package.json` protoc-gen-es 2.16.0+lockfile 在库——生成链三插件坐标齐。
- 锚⑥机检 = **过**：T-0002 闸集已武装、本 commit 过 pre-commit 全闸落地（staticcheck 基线/gofmt/check-lines/env-reads/migration 契约/import 方向闸脚本在 pre-commit 内亲见——staticcheck 需编译 `./internal/...` 兼证可构建）；回填申报 build/vet/gofmt/lines/env/migration/import 全过+`go test` 5 件独立实例绿，宣称件路径实查全对。

注记（非阻断，报管理席知悉）：
- `internal/review` 壳在「六域」命名之外——系票面涉及面钦列件且 master 续存；架构约束 §2「复盘挂账本域」域界口径留管理席裁量。
- `roster.go` BeatLoop `time.NewTicker` 及 client.go Watch `time.Sleep` 退避无 `sched-exempt-ok` 标——收口日（10-23）forbidden_patterns 闸未立（10-24 D-0008 三令终裁后置），心跳/退避语义属豁免径（hostmetrics.go:215 同形已有豁免标先例）；存量行不触闸，后续再编该行建议补注。
- `go test` 实测 DSN=15434 与票面字面 5434 偏差——本机 5434 被 nox-postgres-test 占用如实申报（D-616 合规），独立测试实例语义满足；CI 契约 5434 不受影响。
