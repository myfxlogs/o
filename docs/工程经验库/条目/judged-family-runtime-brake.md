---
schema_version: 2
id: judged-family-runtime-brake
kind: invariant
risk: high
status: verified
keywords: [运行停件, 判官域扩编, 准入判据, killWatch, venue拔线, dm禁用键, 工程径节拍]
scope: [internal/trading, 判官域, dm配置域, 新适配器准入]
summary: 工程径节拍~40min/件不担止血任——判官域新族准入必带运行停件三件套：venue正径拔线+O侧kill消费段+dm禁用键；缺一=要件缺，复审席按条核不落
first_seen: 2026-11-21
last_verified: 2026-11-21
skills: []
evidence_refs:
  - type: doc
    ref: internal/trading/okxgrid/run.go:105
  - type: doc
    ref: internal/trading/okxgrid/verdict.go:53
  - type: doc
    ref: internal/portfolio/hunger/facesnap_whitelist_okxgrid.go:10
---

## 事实/症状

工程径节拍 ~40min/件（check_all→戳→复审→merge→部署）是钱径的
深思熟虑层——刻意不可快化（LTCM 式死法不是修得不够快，是改得
太快没验）。因此钱径紧急纠偏**物理上不能等工程径**：止血必须归
运行面秒级杠杆，不动码不部署。

实证杆齐（okxgrid 族 2026-11-21）：`killWatch` 消费 O 侧 kill 信号
→venue `stop-order-algo` 正径拔线（stopType 钦定回收形，dm
`okxgrid.kill_stop_type` 热调）；判官环 `okxgrid.verdict_enabled`
可禁；`okxgrid.live_switch` 控 live 动作。

缺口：判官域现 okxgrid 包内私有——新族（研究员普查→适配器提案
→判官域）准入判据未成文，复审靠记性是漂移面。

## 修法

判官域新族准入三件套（上线前必带，缺一=要件缺）：

1. **venue 正径拔线**——族方言的 stop/close/deactivate 调用
   （非条件触发，拔线归显式径；okxgrid 先例=stop-order-algo）
2. **O 侧 kill 信号消费段**——killWatch 同构：O 内 kill 语义
   翻译为族拔线调用，消费段败不掩快照段（run.go scan 三段
   隔离先例）
3. **dm 禁用键**——`<family>.verdict_enabled`/`<family>.live_switch`
   形：运行态禁用不卸部署；键域入册（facesnap_whitelist 同构
   登记，漂移探针罩住）

## 判据

- 复审席新族支复审时按本条逐件核三件套——缺一判要件缺不落
- 研究员适配器提案附停件设计为提案要件（普查径出品即带）
- 运行杆本身 buggy 时工程径是唯一径——故杆件代码也走全谱，
  杆的验=check_all 覆盖非免验
- 止血归杆、治病归工程径——两径节拍不同步是特征非缺陷
