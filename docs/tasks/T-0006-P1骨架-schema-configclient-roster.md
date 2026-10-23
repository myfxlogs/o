# T-0006 P1 骨架——schema 底表 + config client + 花名册 + cmd/core + 六域空壳

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 提出 | `[角色:决策]` 2026-10-23 |
| 类别 | 系统骨架（P1） |
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
