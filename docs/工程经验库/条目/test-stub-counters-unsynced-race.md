---
schema_version: 2
id: test-stub-counters-unsynced-race
kind: pitfall
risk: medium
status: verified
keywords: [DATA RACE, 测试桩, stub, atomic, -race, CI红, goroutine, hunger_test, lifecycle, T-0219快径]
scope: [internal/portfolio/lifecycle, 全部 *_test.go 共享桩]
summary: 测试桩计数字段在 Runner goroutine 里 `++`、测试 goroutine 裸读轮询——-race 必报 DATA RACE；本地 check_all 不跑 -race 全绿，CI 一跑即红且被上游 gen 缺失连遮数轮才暴露。桩字段凡被被测件 goroutine 写者一律 atomic.Int64+Load()，禁靠「测试嘛没关系」裸 int
first_seen: 2026-11-07
---

## 病灶

`stubSensor` 8 个 `int` 计数字段（detects/responds/probes/covers/mines/drafts/deps/fstales）：
stub 方法在 `Runner.hungerLoop`/`minerLoop` goroutine 里 `s.detects++`；测试 goroutine
`for stub.detects < 3` 裸读轮询——写读跨 goroutine 零同步，`go test -race` 三件连报
DATA RACE（TestHungerLoopDetectEveryBeat/FirstDetectOnStart/TestMinerLoopFirstMineOnStart）。

## 遮蔽链（为什么现在才炸）

1. 本地 `check_all` go test **不带 -race**——本地永远绿；
2. CI `go test -race -count=1 ./internal/...` 带 race——唯一会咬的闸；
3. 但 CI build 面先被 `internal/gen` 缺失炸（buf.gen.yaml 只含 dashboardv1+scoutv1，
   core 模板没进 gen 步）——build 红在 test 之前，race 病被盖数轮；
4. gen 修复（ci.yml 补 `buf generate --template buf.gen.core.yaml`）后第一次
   跑到 test 步才露馅。

## 对治

- 桩计数凡被被测件 goroutine 写：`atomic.Int64` 字段 + `s.x.Add(1)` 写 + `stub.x.Load()` 读；
- 断言/轮询读点全改 `.Load()`（本次 30 读点两文件）；
- 新写桩先看「这字段谁在哪条 goroutine 写」——被测件 goroutine 写=atomic 起步。

## 实证

- CI run 42379788003：`WARNING: DATA RACE`×4 + lifecycle 包 FAIL；
- 本地 `go test -race -count=1 ./internal/portfolio/lifecycle` 复现→原子化后 6 过零 race。
