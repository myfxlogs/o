# T-0055 账本-venue 对账差根查（~$4.6 缺口）

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` @ 2026-10-26（T-0050 移交项③：recon 告警每帧洪刷 venue 2133.6 vs ledger 2129.0） |
| 领取 | （空=未领） |
| 优先级 | 中 |
| 涉及面 | `internal/ledger/`（recon 面）、`fin_cashflow` 行源、Earn 收益入账径 |
| 范围外 | recon 告警节流（洪刷是症状——先根查缺口） |

## 背景

SG 生产 recon 对账持续告警：venue 权益 2133.6 vs 账本 2129.0，缺口 ≈$4.6。施工席疑 **Earn 收益未入账**（Earn 利息按日结——收益走 bills subtype 进账，若 ingest 未含该 subtype=账本系统性偏低）。

## 验收标准

- 缺口归因实锤（Earn 利息/费用/划转漏类逐类实读 bills vs cashflow 对拍）
- 若=Earn 收益漏入账：bills ingest 补该 subtype 类+历史缺口一次性补记或基线校准
- 若=他因：根因条目化+修复；告警静默=缺口 <$0.5 或入告警白名单（有书面归因才算静默成立）
