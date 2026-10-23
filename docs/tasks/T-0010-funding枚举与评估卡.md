# T-0010 研究域首岗——funding 机会枚举 + 评估卡（探员+验证官上岗）

| 字段 | 值 |
|---|---|
| 状态 | 施工中 |
| 提出 | `[角色:决策]` 2026-10-23 |
| 类别 | 研究域（M1 链 G0→G2） |
| 依赖 | T-0007✅（md_funding 事实面）；T-0008✅（账户面）；`策略评估与仓位`/`okx-market-ws` 技能必查 |
| 领取 | devin-t0002 @ 2026-10-23 |

## 背景

M1 钱径上游：从 `md_funding`/`md_ticks` 枚举报废的 funding 对冲机会（delta 中性：现货多+永续空），产评估卡；**无卡不成提案**（验证官职责卡）。

## 验收标准

- [ ] `migrations/`：`fin_opp_template`（模板书六型先落 funding 型）+`fin_opp_eval` 评估卡表+`fin_proposal` 提案表序进
- [ ] `internal/research/`：枚举器扫 USDT SWAP×funding 率（正 funding→空永续收租向）；净 APR 五拆（毛收−双腿费−滑点−失败成本−窗口损），任一无证据=不可评估禁出卡
- [ ] 评估卡→提案闸：净 APR≥阈（`app_config` 热启）→`fin_proposed` 行；阈值/窗口全机读配置
- [ ] 员工「探员」「验证官」Register+Beat 在岗；产出窗=提案/评估卡行可查
- [ ] 机检全绿；单测覆盖枚举/五拆/闸线（含负 funding 反向、证据缺位拒评两锚）

## 范围外

- 生命周期/风控闸/真单/私有面消费（后续票）；MT5 面；其他五模板枚举（funding 型先通）

## 结果回填

分支 `t/0010-funding-enum`（已推 origin）。施工面=worktree `/opt/phase-wt/t0010`。

**验收锚对账**：

- migrations 三表序进 = **PASS**：`0006` fin_opp_template（venue CHECK+COMMENT）/fin_opp_eval（verdict+direction CHECK 闭集、legs/evidence jsonb）/fin_proposal（open 提案部分索引）+判据种子（五拆参数+min_net_apr+settles_per_day 全 app_config 热启）+funding-hedge 模板行。
- `internal/research/` 枚举+净 APR 五拆 = **PASS**：`funding/{eval,run}.go`——md_funding `DISTINCT ON inst_id` 最新行+25h 新鲜度门（陈行 stale_funding 计门不评估）；五拆 gross=rate×3×365 −（双腿费+滑点+失败成本+窗口损）bps；任一键缺=fail-closed 不可评估（missing_params 门实证）。
- 评估卡→提案闸 = **PASS**：net_apr≥阈→verdict=proposed→`fin_proposal` state=fin_proposed（open 提案去重——同 inst_id 不重挂）；阈/窗全 `research.funding.*` 机读键热启。
- 双员工在岗 = **PASS**：探员/验证官 Register+BeatLoop(30s)；产出窗=fin_opp_eval/fin_proposal 行可查。
- 单测覆盖枚举/五拆/闸线（负 funding 反向+证据缺位两锚） = **PASS**：unit（五拆数值锚/低率闸下/负率反向 rejected 不可执行）+E2E（鲜行提案落卡、负率拒卡、陈行新鲜度门拒评、删判据键 fail-closed+missing_params 计数）全过。

**实测修正**：fin_proposal FK 约束下测试清场须先删提案（E2E 全量跑才现——序位修正）；白名单冻结 research→platform×3 边（13/13）。

**活流关联**：OKX 真流 funding 行已在产（T-0007 采集面）——枚举即吃 md_funding 事实行，无 mock。
