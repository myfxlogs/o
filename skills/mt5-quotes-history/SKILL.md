---
name: mt5-quotes-history
description: mtapi MT5 行情/历史/会话面——Symbols/SymbolParams/GetQuote 族/PriceHistory 族/IsSession 族实测能力与符号方言坑
domain: 工程
---

# mt5 行情历史面（实测 2026-10-17，证据=docs/能力账/mt5-rpc-probe-2026-10-17.json）

| RPC | 功能 | ICR | EXR | XMR | 关键坑 |
|---|---|---|---|---|---|
| Symbols/SymbolList | 全符号表（ContractSize/方言名源） | ok | ok | ok | Symbols 15032313 慢（4.8s）——批量径限频 |
| SymbolParams(Many) | 单符号规格（points/CS/trade_mode/swap） | err（XAUUSD not exist） | err（XAUUSDm 才有） | ok | **硬坑：符号方言**——每 broker 后缀不同（EXR=m 尾实拍），symbol 名一律 venue SymbolList 实读禁猜；生产径走 symMap 原生名 |
| GetQuote/GetQuoteMany | 点价 | err（XAUUSD not exist） | **2026-10-24 复测 ok**（BTCUSDm bid/ask 实回） | ok | 同会话 Subscribe 后活价实返；未订阅孤立径未单测 |
| PriceHistory / PriceHistoryEx / PriceHistoryMany | K 线史 | ok | **2026-10-24 复测 ok**（M1 20 帧） | ok | TimeFrame 枚举 16408=M1；Ex 族用 NumBars 非 To |
| PriceHistoryToday(+Many) / PriceHistoryMonth(+Many) | K 线史 | ok | **2026-10-24 复测 apierr**——「Unable to cast List`1[Bar] to type…」**网关序列化回归**（EXR+demo 双户同现——venue 侧缺陷非方言） | ok（旧） | 族内分面——base/Ex 仍 ok，Today/Month 断；用 Ex/HighLow 绕行 |
| GetTickValueMany | 点值批量 | ok | **2026-10-24 复测 apierr**——「Unable to cast Dictionary`2」**网关回归**（双户同） | ok（旧） | TickValueWithSize 仍 ok——点值走它绕行 |
| IsTradeSession/IsQuoteSession(Many) | 会话内外判定 | ok | err（XAUUSD key miss） | ok | Many 版字段=Symbols 复数；session 表落库换算见 mt_sessions.go（server→UTC） |
| SymbolSessionsEx(Many) | 符号级会话表（报价/交易时段细目） | ok | err（方言同） | err | 同符号方言坑——Many 版入参=symbol 字符串数组 |
| TickValueWithSize | 点值（含 size） | ok | **2026-10-24 复测 ok** | ok | 点值正道=此件；GetTickValueMany 已回归 apierr 见上 |
| TickHistoryRequest/OnTickHistory/TickHistoryStop | tick 级历史（请求→帧流→停） | ok | ok | ok | 请求后 OnTickHistory 流回帧（短窗零帧=窗内无 tick 非故障）；历史补齐面合法拉取 |

**正道锚**：行情判定走 Subscribe+Events 流（D-004）；PriceHistory 族=历史补齐/统计（事件锚定拉取合法）。
**复跑**：同 mt5-account。

<!-- 方法全名索引（proto 名字面 grep 锚）：PriceHistory PriceHistoryMany PriceHistoryToday PriceHistoryTodayMany PriceHistoryEx PriceHistoryExMany PriceHistoryHighLow PriceHistoryMonth PriceHistoryMonthMany -->
<!-- 全名索引2：SymbolParams SymbolParamsMany SymbolSessionsEx SymbolSessionsExMany IsTradeSession IsTradeSessionMany IsQuoteSession IsQuoteSessionMany GetTickValueMany GetQuoteMany -->
