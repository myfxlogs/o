---
schema_version: 2
id: mt5-broker-symbol-trademode-closeonly
kind: pitfall
risk: high
status: observed
keywords: [MT5, trade_mode, ONLY_CLOSE_POSITION, CloseOnly, broker symbol, 执行预检, LEGGED, 单腿裸露, Exness]
scope: [internal/trading/mtexec, execution 双腿构, cfd_perp 开仓径, 预检面]
summary: venue facts 记 Full 不等于 broker 实际 symbol 可开——Exness 平台级 CloseOnly 动态限（账户/时段/区域变），预检按 facts 放行而下单被 ONLY_CLOSE_POSITION 拒，CEX 腿已成交致单腿裸露（NOX posID=468 实证）。
first_seen: 2026-08-31（前身实证） / 2026-10-24（本仓引档）
skills: [mt5-trading-mutating, mt5-quotes-history]
evidence_refs:
  - type: doc
    ref: /tmp/nox/docs/经验库/条目/mt5-trade-mode-canonical-vs-broker-symbol.md（前身条目原文）
---

## 事实（前身实证）
facts/canonical 记 AUDJPY=Full(4)，broker 实际 AUDJPYm=CloseOnly——执行预检放行、ordersend 被 `[code=ONLY_CLOSE_POSITION]` 拒，CEX 腿已成交=单腿裸露。平台 CloseOnly 是动态策略（0 swap 营销等），采集快照不等于下单时可开。

## 处置律（本仓机械锚）
1. **开仓预检=下单时点查 broker 实际 symbol 的 TradeMode**：BuildCfdLegs/下单径对 `SymbolParams.SymbolGroup.TradeMode==FullAccess` 断言，非 FullAccess=fail-closed 拒构——禁读 facts/台账快照当放行证（实勘口径：BTCUSDm=FullAccess 2026-10-24，但动态可变）。
2. **LEGGED 兜底已成**：leg2 拒→leg1 自动平（T-0036 平仓语义根修后真平非反向新开）。
3. **首单前 0.01 手试单 ritual**：venue/账户变更后首批单前以最小手数手动试开即平，验证可开再放自动径——NOX 同款前检纪律。

## 反证条件
若 SymbolParams 当拍回报 TradeMode=FullAccess 而 OrderSend 仍 ONLY_CLOSE_POSITION 拒——则病灶非 trade_mode，转查账户类型/时段/session 闸（参见 session-gate 条目族）。
