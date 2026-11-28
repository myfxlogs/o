---
id: feed-stale-valuation-subscription-gap
title: 订阅集动态化后「估值仍需币」无臂可出+价源顺位读陈——双面同根
date: 2026-11-28
source: 工单 O-41395（票#——NAV 非 USD 腿价源硬化两件同票；BTC-USDT 2026-11-17 tick/index 双死实证，NAV $64,348 vs 实值 ~$60.4k 差 6%）
kind: pit
---

# 动态活跃集的盲区：策略退役带走行情订阅，估值需求没人认领（O-41395）

**坑形（双面同根）**：collector `activeSet`=动态持仓联动册
（`legset.ActiveSetUnion` 多臂派生：提案/策略/dcd uly/basis/
funding-hedge 现货腿/sub_request）。BTC-USDT 的全部需求臂
（dcd 策略、提案、晋级请求）2026-11-17 前后退役→出集→tickers
退订→`md_ticks`/`md_mark_index` 双死。但**估值面仍需该币价**
——`valCcy`/cockpit 敞口内联只吃落库 UNION，无新鲜度判据，
陈价静默入 NAV（差 6% 实证）。两面同根：**「谁需要行情」的
需求枚举与「谁还拿币估值」的持仓枚举是两本账，退役清单只
管前者**。

**根修判别式**：

1. **订阅集是派生物不是注册表**：先判静态册 vs 动态联动册——
   本仓=动态（臂派生）。凡「X 退役后 Y 面仍需其数据」的场景，
   需求源必须**语义并入订阅集**（新臂/新表），禁硬编码币清单
   （清单=第二真源必漂移）。
2. **跨域需求走 bridge-reverse 不绕**：估值币集=fin 域
   （FundCcy∪持仓视图 ccy），PG-md 只见 md|bridge|bridge-reverse
   （T-0187 域禁引律）——按 fin_strategy 先例建正本+projector
   反向镜像+legset 臂三件套；**集合同步语义含出集 DELETE**
   （revTableDiff/reverseSubRequests 均无删径，状态集≠流水集）。
3. **价源判序=鲜度优先非顺位**：fallback 链「先 tick 后 index」
   是顺位不是判据——陈 tick(>24h) 照样赢。正解=**任源 ts
   最新者胜**；REST 现拉 at=now 恒压落库行，但缓存命中须回
   真实拉取戳（陈缓存不冒充鲜）；REST 败回退 DB 最鲜行；
   全无=0 不虚计（现语义保）。
4. **两面同改律**：同一估值语义散在多处（valCcy 函数+cockpit
   内联 SQL）——抽共享件 `ccyPxUSD` 单源消费，SQL 内联面
   改 Go 环（REST 拉取 SQL 表达不了），散修即分叉。

**判别式速查**：读陈价静默入账→先问「数据还在不在更新」
（订阅/任务死没死）再问「读面有没有鲜度判据」；两问皆
否=双面同根一次修。

**连带坑（同票实证）**：新桥表首案撞 `check_bridge_registry.sh`
触发器存在性列——授权列有 `to_regclass` 部署窗 grace，触发器
列漏配同律 grace=「迁移册声明在、实机未部署」即红死锁
（merge 要绿戳、部署在 merge 后）。修法=触发器列补同款
grace（表不存在⇒触发器无从存在，跳；表在触发器缺=照红
T-0194 律不损）。另：bridge-reverse 新表迁移须自带钦定授权
（projector_rw+research_rw SELECT，角色缺席 DO 块跳）——
裸 CREATE 不授=部署后授权列红。
