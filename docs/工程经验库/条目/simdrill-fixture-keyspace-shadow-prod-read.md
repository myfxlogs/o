---
schema_version: 2
id: simdrill-fixture-keyspace-shadow-prod-read
kind: pitfall
risk: high
status: verified
keywords: [演习夹具, venue='sim', 键名空间, 产线盲读, ORDER BY缺, legShapeOf, 派路双实现, 占用格代表]
scope: [internal/trading/simdrill, internal/platform/excell, internal/portfolio/lifecycle, internal/execution]
summary: 演习夹具以产线真键播种 venue='sim' 行——产线 venue 盲读（ORDER BY ts DESC LIMIT 1 无谓词）恒被夹具新行遮蔽=paper 记账断流/毕业闸毒化；修法=夹具键名空间隔离或读面补 venue 谓词+演习毕足迹退役对称件
first_seen: 2026-11-10
last_verified: 2026-11-10
skills: []
evidence_refs:
  - type: commit
    ref: 升件#9581-9583
  - type: doc
    ref: docs/decisions/D-0046-审计臂.md
---

# simdrill 夹具键域遮蔽产线读面

## 坑形（三维同族）

① **夹具真键+盲读遮蔽**：演习夹具 `md_dcd_product`/`md_ticks` 以**产线真键**
（product_id/inst_id）播种 `venue='sim'` 行，产线读面 `WHERE 键=$1 ORDER BY
ts DESC LIMIT 1` 无 venue 谓词——夹具行 ts 恒新恒胜真行：apr=0/expiry 零值
断 paper 记账流、sim 假价毒化现货腿价。产线实证：live dcd 策略
BTC-USDT-261204-66000-P 被同品夹具行遮蔽断流。

② **夹具永生**：夹具行无退役件——逐跑堆叠，遮蔽窗=永久（ephemeral 建/退
对称破）。

③ **查询无序钦定「首见序定」**：excell 占用格代表取查询首见行而 SELECT
无 ORDER BY——堆序即钦定序，同格异器行谁当代表非确定；不可钻异器当代表=
整格凭据 fail→全模板毕业闸红（产线实证 MUU-USDT-SWAP 占格毒化 funding-hedge
格，升件 #9254）。

④ **派路双实现分歧**：单钻径按 inst 后缀派路（-SWAP→okx 双腿）与矩阵径
`legShapeOf(templateRef)` 模板派路双实现——xvenue-carry inst=*-SWAP 落
SWAP 支跑错腿形产 funding 形凭据（错域凭据比缺凭据更毒，毕业闸按
strategy_ref 消费放行）。

## 修法口径

- **产品类夹具=键名空间隔离**：种子 product_id 写 `sim:`+真键（产线读面
  `product_id=$真键` 物理不可见——零产线改码）；钻径消费端 spec.InstID/
  dcdSim 册同键寻址，夹具行与真行同键空间正交。
- **行情类夹具=读面补 `venue<>'sim'` 谓词**：tick 锚须按真器键播种供演习径
  消费（键名空间不可用于腿键链），产线全谱盲读点（paper 现货腿/basis spot/
  eval spot/maker/sizing LatestPx/risk 折 USD/ledger 折算/coverage/depblind）
  统一补谓词——`flow_leg` venue 键域读已是正例。
- **足迹退役对称件**：`retireSimTicks`/`retireSimDcdProducts` 随演习 defer
  清 venue='sim' 行（同 retireEphemeral 律——只清本钻键，他跑行不动）。
- **派路单源化**：`Run` 派路改 `legShapeOf(templateRef)`——注册表外模板
  拒载、shapeXvenue 显式拒载不产凭据、shapeOKXTwoLeg 非 -SWAP 键拒载；
  `simdrill-gate` harness 自体模板须入册否则拒载误伤自体测试。
- **代表选取序定**：占用查询补 `ORDER BY`（state 权重+id 序定——live 优先
  probe 异器殿后），首见序定=显式钦定序非堆序。

## 评审自查

- 新夹具件进仓问三件事：键域隔离否/产线读面谓词否/演习毕退役件否。
- 「同键空间两域行」（fixture×prod）一律按「产线域读面必须域内自洽」审。
- 查询取「首见行」语义处必有显式 ORDER BY，否则钦定与实现相悖。
- 同型前车：`gate-domain-coverage-mismatch`（凭据域与闸域错位族）。
