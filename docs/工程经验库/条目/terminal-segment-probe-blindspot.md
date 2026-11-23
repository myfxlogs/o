---
schema_version: 2
id: terminal-segment-probe-blindspot
kind: pitfall
risk: high
status: verified
keywords: [live_drought, fin_strategy live, fin_position state FSM, 终局段, 漏斗探针, ops_esc_valence, 钱闲置, live_switch, fin_strategy_transition]
scope: [internal/portfolio/hunger 探针族, internal/portfolio/lifecycle 毕业链, ops_esc_valence 惧值册]
summary: 漏斗各段探针全覆盖而终局段（fin_strategy live 在册）零盲——「各段活≠全链活」；fin_position.state FSM 词表无 'live'（open/closing/closed/orphaned），票面判据须锚正主 fin_strategy.state='live'+fin_strategy_transition 末次涉 live 迁移作断流龄。
first_seen: 2026-11-23
last_verified: 2026-11-23
skills: []
evidence_refs:
  - type: ticket
    source: O-35729 / 业主令 2026-11-23②③④
  - type: doc
    ref: migrations/fin_0218_position_state.sql（FSM 词表钉死）
  - type: code
    source: internal/portfolio/hunger/livedrought.go
---

# 终局段无探针 + 判据锚表正位（fin_position 无 'live' 态）

- **坑**：工单#35729（live-drought-firstclass）。hunger 漏斗探针族
  （zero_proposal/open_drought/mono_family/断流沉默依赖盲）各段在册，
  唯独终局段「真钱在赚没有」无盯——fin_strategy live 在册=0 可永久
  静默。业主令「钱闲着=缺陷态与断流/账链断同格」。
- **律①（病灶形）**：分段探针完备≠全链完备——漏斗型系统必须有
  终局段探针，否则中段全绿掩盖终点永空。新探针挂既有拍同槽
  （hungerTick 隔离位族款），esc_class+esc_scope 24h 同窗去重，
  只报不修，归因卡确定性查询先行。
- **律②（判据锚正位）**：票面判据 `fin_position.state='live'` 物理
  不存在——fin_0218 FSM CHECK 钉死 open/closing/closed/orphaned，
  'live' 写入即拒。终局正主=`fin_strategy.state='live'`（live_dud/
  idle_capital 同源判据）；仓位行系 live 策略开仓产物且存活分钟级，
  以仓位计断流=常态误报。**断流龄锚**=fin_strategy_transition
  末次涉 live 迁移 ts（在册=0 时末次必为出 live 降格）；史无 live
  迁移=自 min(created_at) 起算；连策略史都无=按超阈火。
- **律③（惧值固化）**：业主钦定权重随迁移 upsert 进
  ops_esc_valence（ON CONFLICT DO UPDATE 防漂）——live_drought=100
  顶格、idle_capital 55→100、live_dud 75→95、commercial_stall
  75→90、opp_eval_stall 55→85；scope 级键压 class 级
  （ops_esc_pending COALESCE 序）。
- **律④（责任归位）**：追责型升件归因卡末行必载 责任归位
  （钱径管理面 lifecycle/DM）+断点推定+在办证——「谁在让钱闲着」
  卡先答不许空手开会；处置三择一（部署/有理闲置带到期日/问董事会）
  由判词层执行非探针。

## 附：派发钱域加权候选档（裁面——本票只记档不动谓词）

同 P 级元工票与钱域票平权抢席位现状：dispatch 窗内按（级，期限，id)
排无域偏向。候选：工单 note 含钱域词（oss|venue|mt5|eval|live|
funding|kline|钱|账|仓|venue|nav|pnl 等）同级排前。
词表草案（供唤知裁，非钦定）：`money|venue|mt5|okx|eval|live|funding|
kline|账|仓|钱|nav|pnl|现金|持仓|开仓|平仓|保证金|对账|账实`。
动 dispatch_worker 谓词=裁面权，本档仅存证候选。
