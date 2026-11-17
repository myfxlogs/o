---
schema_version: 2
id: simdrill-ledger-venue-domain-not-substituted
kind: pitfall
risk: high
status: verified
keywords: [simdrill, venue='sim', 台账域, legCash, fin_cashflow, venue<>'sim' 谓词, 仿真只替IO, 演习流水污染真钱账]
scope: [internal/trading/execution, internal/trading/simdrill, internal/ledger]
summary: 演习径「全产线同码只替 venue IO」的缝只到 Executor——execution.legCash 以 leg.Venue（路由真相 okx/mt5）落 fin_cashflow.venue，演习现货腿流水以真 venue 名义入账，机器口径 venue<>'sim' 谓词全吞；修法=Runner.WithBookVenue 覆写台账 venue 域（cashflow/slip_sample 归 sim，order_leg.venue 保路由真相）+存量迁移重标
first_seen: 2026-11-17
last_seen: 2026-11-17
evidence_refs:
  - type: ticket
    ref: 工单#22576（D-0048 裁定①，前案 机制#20819 接续）——7/14/30d 机器净产出 -7.15/-5.71/-9.05，剔演习 -0.21/+1.56/-1.78；仓#88/#89 交易所账单 0 条被判词#22350 当真亏
  - type: commit
    ref: fix/simdrill-cashflow-venue-sim（待审）
repro_or_symptom: "simdrill funding/basis 演习跑完，fin_cashflow 多出 venue='okx' 的 spot_buy/spot_sell 行（18+18 对累计 net≈-7.27 USDT）——hunger/commercial_stall/owner_beat/selfaware/death_grad 五处 venue<>'sim' 谓词把演习流水当真钱产出计"
root_cause: simdrill 设计契约是「venue IO 仿真、产线同码」——SimVenue 只替换 Executor/Sizer 双缝，但 execution 落账写的是 leg.Venue（'okx'/'mt5'，路由+语义真相），fin_cashflow.venue 直接继承。venue IO 替了、台账 venue 域没替——「仿真边界画在 IO 层，账本层无 sim 概念」。fin_order_leg.venue 同病但不可同改：guard 平仓腿经 facts.legs 回读 venue 建 CloseLeg 路由执行件，覆写 'sim' 会让 RouterExecutor 找不到执行件=演习平仓径断。
fix_and_prevention: "①execution.Runner 加 bookVenue 域+WithBookVenue 挂接（演习径注入 'sim'），legCash 两 INSERT 与 slipSample 的 venue 列/读件改走 bookVenueOf——域判（spot/合约/mt5 量纲）仍 leg.Venue 不动。②fin_order_leg.venue/execution/position 不覆写：前者=路由真相（平仓腿回读消费），后两表无 venue 列——sim 身份经 strategy_ref→template_ref LIKE 'simdrill%' 可溯。③ledger.daySnapshot 补 venue<>'sim'——快照=真钱域净值锚，夹具域不入（caliber.go 显拒先例同律）。④存量归位=迁移 UPDATE 重标 venue='sim'（关联键=source_table='fin_order_leg'→fin_execution→fin_strategy.template_ref LIKE 'simdrill%'）——勘误冲销留双倍历史毒且验收①口径不消账。⑤凡「仿真缝」设计先问：缝替到哪一层？凡 venue 列落账先问：该列是路由真相还是账本归属域？两者不可混覆写。"
notes: "与 simdrill-fixture-keyspace-shadow-prod-read 互补：那条记夹具行遮蔽产线读面（venue='sim' 键域被盲读吞），本条记写面反向——真 venue 名义的演习流水污染真钱账。venue<>'sim' 已成产线夹具域显式排除的事实约定（md_ticks/md_dcd_product/fin_cashflow 各处谓词在案），新写面入册前先查该约定。"
---

## 演习「只替 venue IO」的缝不到账本层——台账 venue 域须显式覆写

- **实证**：simdrill 全产线同码跑 funding-hedge 演习，开仓 spot 买腿
  `legCash` 落 `fin_cashflow venue='okx'`（leg.Venue 原样继承），平仓
  spot 卖腿同。18+18 行对累计 net≈-7.27 USDT 进了真钱口径——机器
  净产出五处 `venue<>'sim'` 谓词全吞，7d=-7.15/14d=-5.71/30d=-9.05，
  剔演习后 -0.21/+1.56/-1.78。今晨仓#88/#89 交易所账单 0 条被判词
  #22350 当真亏——演习流水伪装真成交的账实背离。
- **教训**：「venue IO 仿真」只保证不碰真钱 venue；台账行的 venue 列
  是**账本归属域**不是执行路由——两层概念必须分开覆写。leg.Venue
  一值两职（路由+落账）是病灶根：路由要真 venue（'okx' 才能派到
  sim executor、平仓腿回读才路由得回），落账要 sim 域。
- **不可同改面**：fin_order_leg.venue 若覆写 'sim'，guard facts.legs
  回读建 CloseLeg.Venue='sim'→RouterExecutor 无 'sim' 键→演习平仓径
  断。xvenue 双 sim（okx-sim/mt5-sim）下 'sim' 单值更是歧义——路由
  真相不可压平。
- **存量处置**：重标优于冲销——venue 列本义=账本归属域，演习成交从
  未触 okx，'sim' 更贴真相；冲销行同样带 venue='okx' 会留双倍历史毒。
- **快照边界**：daySnapshot GROUP BY venue 全量——'sim' 现金行会永续
  产 ('sim','USDT') 日切行；快照=真钱域净值锚，夹具域显式排除
  （caliber.go `venue<>'sim'` 显拒先例同律）。
- **验收锚形**：`venue<>'sim'` 增量差分=0（票面①）+演习一轮后归属
  行全 'sim'（票面③）+修复 sha 为部署祖先（票面④）。
