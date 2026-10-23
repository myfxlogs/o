# skills/ — PHASE 能力账（canon）

> 自 `myfxlogs/nox` master `skills/` @ `b43c02d6` 全量继承（业主定调「skills 随库继承即能力账不重建」）。frontmatter 增 `domain:` 域标签分**工程/经济**；内容未改，文中 `/opt/nox`、nox 仓路径等指前身实物。
> 桥接：`.devin/skills` → `../skills`（app 侧只加桥不搬件，D-608 规）。

## 域分

### 工程（20）

**venue 面能力账**（端点实测+坑，触 venue/API 面必先查）：

| 域 | skills |
|---|---|
| binance | `bn-market` `bn-spot-account` `bn-fapi-account` `bn-fapi-market` `bn-sapi-wallet-earn` `bn-userdata-ws` |
| mt5 | `mt5-gateway-admin` `mt5-account` `mt5-quotes-history` `mt5-orders-streams` `mt5-trading-mutating` |
| okx | `okx-market-ws` `okx-private-ws` `okx-rest-boundary` `okx-exec` |
| pg | `pg-capacity` |

**规程面**（强制工作流类）：

| skill | 触发面 |
|---|---|
| `生产取证` | 断言运行时状态前必走（SSH→docker logs→psql→venue 实拍三通道） |
| `协议面取证` | 对外部 API/gRPC/流下「无某能力」结论前必走 |
| `告警分诊` | 告警→AI 分诊席 runbook |
| `阶段交付` | Phase/关键修复收口格式 |

### 经济（1）

| skill | 触发面 |
|---|---|
| `策略评估与仓位` | 策略评审/淘汰裁定/仓位调整前必走——统计判据（置信界/依赖性检验）+ 仓位规模方法（分数 f/几何阈值/保证金上限/相关性输入）。方法源 Ralph Vince《The Mathematics of Money Management》中译 `前身路径 /opt/nox/MathematicsMoneyManagement_zh.md`——**该文件未随搬，升格待取料** |

## 使用纪律

- 触及 venue/API 面的开工指令必带对应 skill 指针；有而未查=复审退回理由（总纲 D-617）
- skill 在用中被证错/缺面→同单内修件是职责（D-641），不推后
- 证据指针回指 `docs/能力账/`（nox 侧）——PHASE 能力账档未建，落位走任务单
