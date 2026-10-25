---
schema_version: 2
id: venue-holding-surface-blind-spot
kind: pitfall
risk: high
status: verified
keywords: [venue 钱面完整性, dcd 双币赢, 结构性持仓, balance 盲区, NAV 漏算, order-history, OKX]
scope: [cmd/core NAV/卡面/对拍, internal/ledger FundCcy, venue 钱面盘点]
summary: venue 产品型持仓（OKX 双币赢 dcd/结构化产品）不出现在 balance/positions/savings/staking-defi 任何标准余额面——$2.4k BTC 覆盖卖 call 单锁仓 65 天三面全盲（NAV/卡面/对拍）；venue 钱面盘点必须按产品族枚举（order-history 系），余额面不等于钱面
first_seen: 2026-10-25
last_verified: 2026-10-25
skills: [okx-asset-funding]
evidence_refs:
  - type: server_fact
    ref: "生产实探 2026-10-25：positions=1（仅 SWAP 腿）、OPTION/挂单/bills=0，dcd/order-history 返 LIVE ordId=11448262 BTC-USDT-261225-95000-C notional 0.0207 BTC ≈$2,400"
  - type: doc
    ref: docs/tasks/T-0101-双币赢持仓入账.md
  - type: doc
    ref: https://www.okx.com/docs-v5/zh/#financial-product-dual-investment
---

# venue 持仓面盲区——产品型持仓三面全盲（T-0101 实证）

## 病灶

业主报「OKX 还有 1 个 ETH call 持仓」→ 决策席全谱探：positions/挂单/期权史/
子账户/bills 全零 → 初判「不在基金户」。业主再指「双币赢 BTC-USDT」→
`/finance/sfp/dcd/order-history` 实拉：**LIVE 0.0207 BTC ≈$2,400**——
标准余额面全族不可见：

| 读面 | dcd 持仓可见性 |
|---|---|
| `/account/balance`（details 逐币） | ❌ 锁仓本金出余额 |
| `/account/positions` | ❌ 非持仓语义 |
| `/asset/balances`（资金户） | ❌ |
| `/finance/savings/balance`（Earn） | ❌ 独立产品族 |
| `/finance/staking-defi/*` | ❌ 族不同 |
| **`/finance/sfp/dcd/order-history`** | ✅ 唯一枚举面 |

$2.4k 基金资产对 NAV/钱在哪/对拍三面隐形 4 天（10-21 创建早于账本基线
10-24——基线校准把洞口一并吸收未现形）。

## 修法（T-0101 承接）

venue 钱面=**余额族 + 产品族**双枚举：余额面之外按 venue 产品目录逐族拉
持仓面（OKX：savings、staking-defi、sfp/dcd、fixed-loan…；新族上线=盘点
义务非可选）。账本侧在持产品=已部署资本入 FundCcy 第三类别。

## 教训

- 「余额读全=钱面全」是伪等价——**产品持仓出余额枚举是 venue 通例**
  （锁定即出账）。任何 venue 接入/钱面盘点票，产品族枚举为必备步骤。
- bills 账单的认购 subType（OKX：160-163/207-210 双币赢族）是第二探针面
  ——余额缺口的另一半证据在流水侧。
- 业主一句话「还有 1 个持仓」值一张全谱探针工单——第一遍全零结论不可
  收，追问产品形态（call 订单→双币赢结构化单）后才现真面。
