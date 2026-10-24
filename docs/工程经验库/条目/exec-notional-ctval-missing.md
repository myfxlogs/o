---
schema_version: 2
id: exec-notional-ctval-missing
kind: pitfall
risk: high
status: verified
keywords: [ctVal, 名义额, 张数, notional, swap, 漂移误判]
scope: [cex 合约腿, okx swap, 名义额/净敞口计算]
summary: 合约名义额=sz×ctVal×px——sz 是张数非币量，漏 ctVal 因子错一个数量级（假漂移误诊）
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: [okx-api-spec, okx-exec]
---
# exec-notional-ctval-missing —— 合约名义额 = sz×ctVal×px，sz×px 错一个数量级

## 坑
合约腿名义额用 `sz×价格` 计算是**错一个 ctVal 因子**——`sz` 是张数不是币量。0.03 张 BTC-USDT-SWAP 被算成 0.03×80700=$2421（真实名义=0.03×0.01×80700=$24.2），对现货腿产生 97% 假漂移。

## 实证
PHASE 仓 1（2026-10-24 真户）：guard `notionalNow` 用 `sz×last`，SWAP 腿 0.03×80700=2421 vs 现货腿 24.9 → delta_drift 19587bps 误触发平仓级联（真实两腿名义 ~$24.2 vs ~$24.9，漂移 ~285bps 仍超 20bps 阈值——**数学错了结论碰巧对，但触发理由全错**）。级联最终 LEGGED+人工介入。

## 反证条件
- 任何名义额/敞口/保证金计算：合约面必走 `sz×ctVal×markPx`（正向/U本位）或 `sz×ctVal`（反向/币本位）——ctVal 从 instruments 实读，禁按币种名猜
- 现货/CFD 各自有自己的量纲（现货=币量、MT5=lots×contractSize）——统一名义额函数必须按乐器类型分派
- 同族：cex-market-buy-sz-quote-ccy（sz 计价币口径）、T-0037 ctVal 币本位

## 机械化
- T-0046：guard notionalNow 按乐器类型分派（SPOT=sz×px；SWAP/FUTURES=sz×ctVal×px），乐器元数据决策时点现读；漂移判据用统一名义口径。
