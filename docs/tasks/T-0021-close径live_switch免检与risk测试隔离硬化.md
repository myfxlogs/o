# T-0021 close 径 live_switch 免检 + risk 测试隔离硬化（T-0015 缺口两枚）

| 字段 | 值 |
|---|---|
| 状态 | 完成 |
| 提出 | `[角色:决策]` 2026-10-23（T-0015 施工上报两缺口，均属实裁立） |
| 类别 | 缺陷修复×2（钱径安全+测试面） |
| 依赖 | T-0015✅ |
| 领取 | devin-t0002 @ 2026-10-23 |

## 缺口一：live_switch 拦平仓=存风险（钱径级，先修）

T-0014 实现的 `Execute` 对 `invest.lifecycle.live_switch` 一律 fail-closed——close intent 同走此闸意味着**委员会关闸时有仓退不掉**，与「拦平仓=存风险」原则（kill_switch 不拦 close 同款逻辑）相抵。

- 裁：**live_switch 是开仓闸非退仓闸**——close intent 免检 live_switch（仍走令牌+留痕）。`Execute` 判序加 intent kind 分流：open 查 live_switch，close 跳过。
- 测试锚：live_switch=false 下 open 拒执（原锚不破）+ close 放行（新锚）。

## 缺口二：risk 测试共享库并发脆性（既有病灶）

master 裸跑复现：`TestAllowDenyAnchors`/`TestTokenMechanics` 在全量并发下随他包 fixture 数据态翻（venue 快照/敞口行被他包顶过阈值）；ledger 锚持存 `execution_id=1` 假设。

- 修：risk 测试改独立 fixture 命名空间（专属 inst_id/strategy_ref 前缀+判据阈远离他包值域）或 t.Run 隔离清场；ledger 锚去持存 id 假设。
- 锚：`go test -count=3 ./internal/risk/...` 与全量 `go test ./...` 双语境三连绿（CI 面同证）。

## 验收标准

- [ ] close 径 live_switch 免检（双锚）；open 径原判不破
- [ ] risk/ledger 测试隔离硬化；全量三连绿+CI 绿
- [ ] 两坑各入工程经验库条目（D-585 修复票必填；缺口一属「闸语义把退出当进攻拦」——值得独立条目）
- [ ] 机检全绿

## 范围外

- 看护其他判据；live_switch 语义本身的委员会流程细化（P4 面板期）

## 结果回填

分支 `t/0021-close-gate-testiso`（已推 origin）。施工面=worktree `/opt/phase-wt/t0021`。

**验收锚对账**：

- close 径 live_switch 免检双锚+open 原判不破 = **PASS**：Intent.Kind（open 缺省|close）判序分流——close 跳过 live_switch（令牌照验不松）；测试锚 live off 下 open 拒执+close 放行双腿成交。
- risk/ledger 隔离硬化+三连绿 = **PASS**：testutil.Pool 每包独占 `t_<pkg>_<rand>` schema+AfterConnect search_path（治本——guard 翻 kill_switch 不再误伤 risk，跨包 fixture 零干扰）；ledger 去持存 execution_id=1 假设；risk 专属 fixture 名空间+venue='rktest' 双保险；`go test ./...` 三连全绿 13 包×3 实拍。
- 双坑入库 = **PASS**：`gate-semantics-exit-vs-entry`（critical——闸拦进攻非退出，拦平仓=存风险）+`test-shared-db-namespace`（high——本地绿≠隔离绿，修法=隔离层非断言放宽）schema v2 齐。
- 机检全绿 = **PASS**：build/vet/gofmt/lines/env/migration/import 全过。

**实测修正**：testutil schema 隔离=一处修全域受益（risk/ledger/guard 三包病灶同根同源——非逐测补桩）。
