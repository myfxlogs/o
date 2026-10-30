# T-0011 okxpriv E2E 竞态修复（CI 回归红）

| 字段 | 值 |
|---|---|
| 状态 | 审计通过 |
| 提出 | `[角色:决策]` 2026-10-23 |
| 类别 | 缺陷修复 |
| 领取 | devin-t0002 @ 2026-10-23 |

审计席：headless @ 2026-10-31

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

## 无头审计判决（审计席：headless @ 2026-10-31 —— **通过**）

收口链：施工 a145f52a → 回填 7fcc50be → 验收合入 28fa82da/98f05bf5。逐锚对账：

- **锚一 CI 绿+TestPrivE2E 稳定（本地+CI）= PASS**：合并头 98f05bf5 携 `build-test`
  check=success（gh api commits/check-runs 实证 @2026-10-23T14:07Z），该步跑
  `go test ./...` 全谱；当时 ci.yml 注入 `PHASE_TEST_PG_DSN`@5434 与
  `testutil.Pool` 读键同号——测试在 CI 真跑非静默 skip，套件绿=本测试过。
  本地三连宣称与机制自洽（ws 经 hijack，srv.Close 不待桩内 5s 挂住，0.17s/拍可信）；
  后续 7 日全史无本测试复发条目（flaky 条目三件皆别域），稳定面成立。
- **锚二竞态根因入经验库 = PASS**：`docs/工程经验库/条目/ws-e2e-stub-handshake-race.md`
  在档（schema v2 frontmatter 齐：id/kind/risk/status/keywords/scope/evidence_refs 全字段），
  三层根因+三件同修+判据实录，现行 `索引.md` 已含行（gen 机生面收录）。
- **锚三机检全绿 = PASS**：CI build-test 链（build/vet/gofmt/lines/import/test）success
  即当机机检权威面；禁用原语闸（forbidden_patterns）建于 10-24——本票收口 10-23，
  `time.Sleep` 测试件新增行不溯及，无违例。

**修法正确性复核**：桩回 `{"event":"login","code":"0"}` ack 与 T-0008 生产语义（先 ack 后订阅）
对称=真根修非症状掩；断言 deadline 轮询替固定睡眠=就绪条件语义；inst_id/ccy 专属锚+用前清场
=残留假绿面封死。现行 master 测试件已经 T-0048/T-0089/T-0167 演进（快照表 DROP 后断言改查
md_account_events）——演进链与本案不悖。
