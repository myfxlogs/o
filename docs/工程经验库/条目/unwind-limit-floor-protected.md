---
schema_version: 2
id: unwind-limit-floor-protected
kind: strategy
risk: high
status: verified
keywords: [回卷, 平仓, 清场, 限价保护卖, floor价, 薄盘, 滑点, unwind, 事故处置]
scope: [一切仓底回卷/清场/事故处置, spot/perp 通用, okx, mt5, 薄盘币]
summary: 回卷大额头寸用「限价卖钉在 bid×(1-δ)」替代市价裸砸——吃得到即市价速度成交、吃不到挂着不亏下破，零下破风险；薄盘票唯此径
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: [okx-exec, okx-account]
evidence_refs:
  - type: doc
    ref: tools/okxunwind/main.go
  - type: doc
    ref: docs/工程经验库/条目/probe-restore-scope-overreach.md
---

## 事实/症状

PENGU 意外重仓 $2145 回卷：book 浅（20 档 bid 才盖住全量）。若市价一单砸出=按深度逐档吃薄，滑点不可控。改限价卖 floor=best bid×0.998——单轮 278,765 张全成，成本 ≈0.33%（点差+费），零下破。

## 根因或策略解释

限价单在 bid 下方挂出=「taker 速度 + floor 价保护」双得：对手方买单深度够则立即成交（同市价），不够则剩余量挂在 floor 价=不破防。市价单没有这个底——价格风险敞口到 book 最薄档。

## 处置

- 回卷/清场/事故处置默认径：`px=bid×(1-δ)`（δ≈0.2%~0.5% 按簿厚调）限价卖，批间实读余量，挂住未全成→撤单重挂贴近新 bid
- 尾量 <minSz=不可交易残尘，记实保留（禁为清零凑单）
- 对称买侧同理：`px=ask×(1+δ)`

## 证据

- check: okxunwind -ccy PENGU 真户跑
- result: r1 limit sell 278765.1856 @0.0076476 一轮 filled；Earn USDT 2132.91 复位
- ref: 2026-10-24 事故回卷实录

## 反证条件

- check: 某 venue 限价单低于市价被当 maker-only 拒/不入簿
- falsifies_when: venue 拒收低于 bid 的卖限价（post-only 强制场）——届时退化为分批市价+簿深预扫

## 适用边界

spot/perp 通用；极端插针行情 floor 要放宽或换分批市价。仅适用「退出有底线的处置」，不适用于抢价成交需求。
