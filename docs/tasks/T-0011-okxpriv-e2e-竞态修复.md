# T-0011 okxpriv E2E 竞态修复（CI 回归红）

| 字段 | 值 |
|---|---|
| 状态 | 待验 |
| 提出 | `[角色:决策]` 2026-10-23 |
| 类别 | 缺陷修复 |
| 领取 | devin-t0002 @ 2026-10-23 |

## 背景

T-0008 合入后 CI `TestPrivE2E`（internal/data/okxpriv）红：`collect_test.go:66 事件未落事实行 n=0`。本地 15434 过、CI 炸——疑似 T-0008 回填所述「collector Register 遇 ctx 取消静默返回」竞态同族残留。

## 验收标准

- [ ] CI `go test ./...` 绿；`TestPrivE2E` 连续 3 跑稳定过（本地+CI）
- [ ] 竞态根因写入工程经验库条目（D-585：修复票坑处置必填）
- [ ] 机检全绿

## 范围外

- 功能扩面

## 结果回填

分支 `t/0011-okxpriv-e2e-fix`（commit a145f52，已推 origin）。

**验收锚对账**：

- CI 绿+三连稳定 = **本地 PASS / CI 待远端跑**（CI 触发面=push master/PR，决策验收合并后自动证；本地 TestPrivE2E 连续 3 跑全 PASS 0.17-0.18s）。
- 竞态根因入经验库 = **PASS**：`docs/工程经验库/条目/ws-e2e-stub-handshake-race.md`（schema v2 frontmatter 齐；三层根因+三件同修修法+判据在案）。
- 机检全绿 = **PASS**：build/vet/gofmt/lines 全过；`go test ./...` 8 包全绿 @15434。

**根因实录**（本地干净库复现 CI 红 10s 超时后钉死）：
1. 桩握手不对称——T-0008 login-ack 先行修复后，fake 服务端仍不回 ack→collector 同步等 ack 挂死；
2. 固定睡眠断言——`Sleep(1500ms)` 按时序假设非就绪条件；
3. 共享测试库残留假绿——通用键断言被历史残留行秒过掩盖前两层。
**修法三件同修**：fake 回 login ack / 断言改 deadline 轮询 / 专属标记锚+清场。
