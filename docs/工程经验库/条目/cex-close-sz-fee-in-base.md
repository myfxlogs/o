---
schema_version: 2
id: cex-close-sz-fee-in-base
kind: pitfall
risk: high
status: verified
keywords: [平仓量, availBal实读, taker费基准币扣, LEGGED, 委托量非持仓量, sz]
scope: [cex 现货平仓, okx, 平滚径, 一切「再操作量」读径]
summary: 现货平仓量禁抄腿行 sz——taker 费从基准币扣实持<记录；一切「再操作量」必须 venue 余额实读现况推，禁从当时委托参数推
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: [okx-exec, okx-account]
---
# cex-close-sz-fee-in-base —— 现货平仓量禁抄腿行 sz：taker 费从基准币扣，实持 < 记录

## 坑
现货买入腿的 `sz`（委托量）≠ 实际持仓量——taker 费**从收到的基准币里扣**（买 0.00030897 实收 0.00030866）。平仓照抄腿行 sz 卖 0.00030897 = 超持仓拒单 → LEGGED。

## 实证
PHASE exec5 leg2（2026-10-24 真户）：腿行 sz=0.00030897 记录委买量，venue fills 实证 fee=-0.00000031 BTC 从实收扣 → 实持 0.00030866。平卖 0.00030897 被拒（venue_err 未留痕——另一缺陷），腿1已平、腿2残 → LEGGED+回滚歧义+人工介入。

## 反证条件
- 现货平仓量=**决策时点 venue 余额实读**（availBal）∩腿行记录——D-0008 钦定径；禁抄委托量
- 合约腿平仓 sz=张数不受此病（持仓数自 venues 读，非委托记录）
- 通用律：**任何「再操作量」都从 venue 实读现况推，不从当时委托参数推**——委托≠成交≠持仓

## 机械化
- T-0046：closeLegOf 对 cash/spot 腿改读 venue availBal（取 min(leg.sz, avail) 网格对齐）；合约腿 sz 读 positions。
