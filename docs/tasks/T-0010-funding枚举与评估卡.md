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
