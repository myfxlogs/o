---
schema_version: 2
id: nav-attribution-three-buckets
kind: rule
risk: medium
status: verified
keywords: [NAV, 对账, 账房, books_daily, nav_residual]
scope: [cmd/core/books_beat.go, md_equity_curve, fin_venue_bill, fin_cashflow]
summary: 净值跳变先拆三桶（净流+盈亏+残差）再下判词——「涨=钱进来」与「涨=赚的」是两回事；残差超阈=账面闹鬼（快照缺腿/离线进出/未载费类）升件，11-09 $4,782 幽灵跳水即首例实证
---

## 规则

净值跳变 **永远先拆三桶再下判词**：

```
ΔNAV = 外部净流(fin_venue_bill deposit+withdraw)
     + 盈亏(fin_cashflow 六族,同 profitDailyBeat 钦定口径)
     + 未释残差
```

- |残差|>阈(现 $200) = **账面闹鬼**——快照缺腿/离线进出/未载费类，定性归会话
- 残差≈0 时「净值涨=钱进来」和「净值涨=赚的」是两件事，混读=自骗
- 判「赚没赚」用盈亏桶，不用 ΔNAV——Δ 被资金进出主导

## 实证锚（首例鬼账）

2026-11-09 11:09 UTC 总 NAV $15,870→$11,087（-$4,782），fin_venue_bill 同窗
零出入行；11-11 11:43 自回 $16,084。三桶拆：净流 0 / 盈亏≈0 /
**残差 -$4,782**——快照腿缺（疑 Earn/venue 腿漏拍），非真亏损。
若当时无人手工对账，此鬼账永不见天——账房器官即此例制度化。

## 口径坑

- md_equity_curve `ccy='NAV'` 才是总账行；`venue='ledger'` 下按币分腿
  同名混排，**勿按 venue 拆腿**（v1 腿差归因因此略）
- fin_venue_bill 无 venue 列=okx 单源；mt5 进出无 bill 源，
  其动静天然落残差桶——读残差先记此偏差
- 内部划转 type=1 不入净流（账户间搬运不改总 NAV）
