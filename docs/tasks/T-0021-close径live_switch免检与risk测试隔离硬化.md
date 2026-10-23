# T-0021 close 径 live_switch 免检 + risk 测试隔离硬化（T-0015 缺口两枚）

| 字段 | 值 |
|---|---|
| 状态 | 施工中 |
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
