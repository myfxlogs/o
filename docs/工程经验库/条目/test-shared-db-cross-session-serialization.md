---
schema_version: 2
id: test-shared-db-cross-session-serialization
kind: pitfall
risk: medium
status: verified
keywords: [o_test 争用, pg_advisory_lock 860223, 跨席全测, 超时假红, go test timeout, 管道吞退出码]
---

# 跨席并发全测打同一 o_test——迁移锁串行化放大成超时假红

## 现象

两施工席同时 `go test ./...`（同一台开发机、同一个 o-postgres-test:5434）：
每包每个 testutil.Pool→`pgstore.Migrate` 都过 `pg_advisory_lock(860223)`
（pgstore/migrate.go 钉的单例迁移锁），两席套件逐锁串行——单包 solo
9min（cmd/core）被放大到 25min+，撞 `-timeout` 上限即 panic
「test timed out」，形似代码红实为基建争用。

## 实证

- T-0170 施工席 2026-10-27：首跑给 `-timeout 540s`，cmd/core 9m0s 整
  panic 于 TestListCashflows（solo 独跑 553.9s 绿——纯预算错）。
- 复跑 `./... -timeout 1500s` 期间另一席 `go test ./internal/research/... ./cmd/...`
  同点开跑（ps 双 go test 实拍）：`pg_locks` 实拍 advisory 860223
  1 持+10 排——套件拖到 40min+：3 包撞 1500s 帽（cmd/core/earn/guard）。
- **判红两段法**：他席排空后嫌疑包独跑——earn/guard 转绿（证超时帽=
  争用）；sigdist TestBasisAnn/lifecycle 两案独跑仍红——再取净
  origin/master 树对拍同值红 → **存量红非争用非本票**（两段缺一
  会误判：独跑红≠你引入，净树对拍才定谳；breakeven 案与
  test-dsn-silent-skip 条目 T-0124 旧案同形——夹具漂复发面）。
- 二坑叠加：`go test … | grep -v ok | head` 管道让 `$?` 变 head 退出码
  0——panic 被假绿掩盖（collab §11「禁管道吞退出码」同律，测试面同样适用）。

## 修法/防法

- **涉面两包绿即施工测闸达标**：改动面包（如 cmd/core+cmd/research）
  独跑带 DSN 绿=该席测闸；全树 `./...` 印记归 CI（分支 push 后权威跑），
  不在共享测试库上跨席抢跑全树。
- 必须跑全树时：先 `ps aux | grep 'go test'` 探他席在跑否；跑测命令
  `set -o pipefail` 或落日志后 grep，禁管道尾命令冒充退出码。
- 超时预算按「solo 实测×并发席数」给（cmd/core solo 554s——两席并发
  预算 ≥30min/包），panic 先查 `-timeout` 与并发再疑代码。

## 证据

T-0170 施工回填（测试证据段）；ps/pg_locks 实拍 2026-10-27（双 go test
进程+860223 排队）；solo 553.916s 绿 vs 并发 25m panic 对照。
