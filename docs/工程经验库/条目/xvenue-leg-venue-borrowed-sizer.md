---
schema_version: 2
id: xvenue-leg-venue-borrowed-sizer
kind: pitfall
risk: high
status: verified
keywords: [xvenue, guard, sizer, venue路由, mt5, 平仓量, 跨所]
scope: [guard, execution, xvenue, close]
summary: 跨 venue 双腿平仓量决不许借 venue 现读件——mt5 腿抄 okx sizer=异 venue 误读开仓险
first_seen: 2026-11-08
last_seen: 2026-11-08
evidence_refs:
  - type: ticket
    ref: T-0145
---

## 跨 venue 双腿：venue 现读件必须按腿 venue 路由，缺件=fail-closed 不兜底

- **实证**：guard 平仓量 VenueSizer 原为「单 sizer 兜底任意 venue 腿」——mt5
  CFD 腿若未挂 venue 路由件会落回 okx sizer 调 ContractPos("BTCUSDm")=
  异 venue 目录查无此乐器，读数不可信仍照样构平仓腿（错 venue 件=错读量
  开仓险）。T-0145 桩测「venue 无路由应 fail-closed」红桩当场照出。
- **教训**：venue-keyed 读面（sizer/dialect/凭据）一旦多 venue，「缺路由
  兜底用主 venue 件」看似省事实为异 venue 误读通道——正确形态=按腿 venue
  分派+外 venue 无专用件硬拒（okx/空仅许 okx 默认件兜底）。
- **关联病灶**：腿行 venue 字段被父腿单值抄送（legSet.venue 单字段）同构
  ——跨 venue 双腿任何「共享父值」字段都要按腿取（venue/ticket/sz_unit）。
