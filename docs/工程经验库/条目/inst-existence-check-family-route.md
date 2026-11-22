---
schema_version: 2
id: inst-existence-check-family-route
kind: pitfall
risk: high
status: verified
keywords: [inst 存在性核验, 族路由, dcd 双币赢, 51001, 幻影退役, venue 不存在判词, OKX]
scope: [cmd/core open/loadMeta, internal/trading/okxexec InstFamily, venue 存在性核验判词]
summary: 「venue 不存在」判词的前提=核验打对了 API 族——DCD 产品键（base-quote-YYMMDD-行权价-C/P）在 sfp/dcd 产品册不在 /public/instruments 衍生品仪器册，跨族查空=结构性假阴性；287249 策略被以「inst 51001 不存在」幻影退役实案（仓42/单11594773 实为 LIVE 真仓）
first_seen: 2026-11-17
last_verified: 2026-11-17
evidence_refs:
  - type: server_fact
    ref: "ops_event#22918 处置席裁定：唤知幻影判词双错——/public/instruments 查错面（DCD 产品在 sfp 产品册非衍生品仪器册）+/account/positions 对 sfp/dcd 结构性恒空≠无敞口；287249 退役已执行于伪前提但结局仍立"
  - type: server_fact
    ref: "ops_event#33341：ordId=11594773 SETTLED 回款实证+LIVE 单在场；仓42=该单 productId 覆盖源"
  - type: doc
    ref: docs/工程经验库/条目/venue-holding-surface-blind-spot.md
---

# inst 存在性核验族路由——跨族查空≠不存在（287249 幻影退役案）

## 病灶

2026-11-17 唤知会话核仓42「幻影」：venue positions=[] + inst 查
`/public/instruments` 得 51001/空 → 判「inst venue 实证不存在」→
策略 287249 退役 + 工单 phantom-pos-42-dcd-usdt。处置席复核双错：

1. **查错面**：`BTC-USDT-261204-66000-P` 是 OKX SFP/双币赢产品键，
   在 `/finance/sfp/dcd` 产品册（order-history/order-status），
   不在 `/public/instruments` 衍生品仪器册——查 instruments=结构性
   恒空，空≠不存在。
2. **读面误用**：`/account/positions` 对 sfp/dcd 产品结构性恒空
   （产品型持仓不入标准仓位面，同 venue-holding-surface-blind-spot）。

真相：ordId=11594773 venue order-history=LIVE（后 SETTLED 回款
+0.026 USDT），仓42=真仓非幻影。退役前提伪、结局碰巧立（业主本就
不想要该策略）——**伪前提的正确结局是运气不是判据**。

## 判据（钉死）

- 「venue 不存在」判词行**必须携所查族标识**（`[核验族=X]`）——
  不带族标的不存在判词=前提不可复核，禁落。
- 键形→族单射路由：`*-P/*-C`+YYMMDD 日期戳+DCD 结构 → sfp/dcd 面；
  `BASE-QUOTE_SETTLE-YYMMDD-行权价-C/P`（带 _UM/_U settle 段）→
  OPTION 族 instruments；`-YYMMDD` 尾 → FUTURES；`-SWAP` → SWAP；
  `BASE-QUOTE` 两段 → SPOT；其余 → unknown 不许猜族。
- 码面落实：`okxexec.InstFamily`/`DcdProductKey`/`FamilyCheckNote`
  （internal/trading/okxexec/instfamily.go）；`loadMeta` 对 DCD 键
  显式族标错误上抛不落 instruments 族静默查空。
- 采集面同根：模板禁用≠传感器死——`md_dcd_product` 枚举面声明闸
  摘 `t.enabled`（目录=venue 真相，禁因复核需新鲜目录证伪）；
  提案闸在模板态对称补半（`persistEval` tplOn——禁用模板评估卡
  照落 proposed 不起提案）。

## 教训

「查无此物」之前先答「我查的是哪个族的册」——venue 产品族各有
独立册面（instruments/order-history/products），跨族查空是
API 形态的结构性假阴性，不是证据。
