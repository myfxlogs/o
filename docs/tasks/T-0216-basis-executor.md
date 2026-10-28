# T-0216 basis 族 executor+short 腿能力缺口

> 状态：施工中 | 领取：devin-t0216 @ 2026-10-28 | 优先级：P2（次于 T-0215——数据刚就位，executor 后一步）
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
