# T-0216 basis 族 executor+short 腿能力缺口

> 状态：施工中（返修） | 领取：claude-glm-t0216r @ 2026-10-29（首交 devin-t0216 打回返修续任） | 优先级：P2（次于 T-0215——数据刚就位，executor 后一步）
> 立项：决策席 @ 2026-10-28

## 面况

- quarter_basis_long/perp_quarter_spread 两腿全在能力账
  （trade-order 覆盖 spot/perp/futures）——评估器修完（T-0213）即
  可评，executor 是开仓最后一块。
- quarter_basis_short 需 `spot_short` 腿=保证金借币卖出——能力账
  无 margin borrow 面（OKX /api/v5/account/borrow-repay 未在
  fin_capability_surface 在册）→条件缺口件，先立 leg 能力再谈
  executor。

## 钦定修法

- executor 钉版后接 funding-hedge 开闸同构（持仓/平仓/ledger 全链）。
- paper 径先行同 T-0215。
- spot_short 腿：margin borrow 能力面立项补齐（endpoint 在册+
  读面实测）→不入账=quarter_basis_short 恒缺腿评不了一条腿。


## 结果回填

**实现**（六件）：①`grammar.go`——quarter_basis_long/perp_quarter_spread
Executor 落钦定名；quarter_basis_short NeedFace 挂 margin-borrow-repay
（enabled=false→族不枚举=恒缺腿如实呈，不再产假想卡噪）；②evalBasis
proposed→fin_proposal 提案径（evidence 补 apr 键过 T-0205 硬闸，inst=季度
期货键 direction=basis_quarter）；③lifecycle paperTemplates+两族（short
不入——腿缺）；④`execution.BuildBasisLegs`——衍生腿=到期日期货 sell
（FUTURES 乐器域 ctVal 张数径）+多头腿 spot quote_ccy/perp 张数，futUSD
反推对齐律同 T-0062；⑤`cmd/core/basis_open.go` 派发——cap/冷却/闸/腿构/
Execute 原子双腿全链 funding-hedge 同构，loadMeta -YYMMDD→FUTURES；
⑥`md_1061_margin_borrow_capability.sql`——borrow-repay（write）+
borrow-repay-history（read）双行在册 enabled=false（登记表补行）。
测试：`TestBuildBasisLegs` 绿（USD 面值 10 万张 fut+spot quote_ccy $1000
对齐/perp 币本位 1.23 张）；假想卡断言随 executor 落位移锚 ≥3→≥2。
