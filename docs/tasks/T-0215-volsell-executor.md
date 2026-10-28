# T-0215 vol-sell 族 executor——strangle/single_call 从评估到钱径

> 状态：待施工 | 优先级：P1（唯一「数据齐+评估通+能力在账」的新族）
> 立项：决策席 @ 2026-10-28

## 面况

- `md_dcd_product` 8094 行鲜（<30min）——evalVolSell 已过
  product_rows_missing 关，下一拍起恒落 `rejected/executor_missing`
  （设计语义：面在可评，executor 缺）。
- DCD 写径已修（T-0207 quote→trade→status 方言+T-0208 ambiguous/
  venue_err）——executor 复用 dcd_call/dcd_put 现役径非从零。
- 假想卡 id=6/7 要求 executor 名 `vol_sell_strangle_executor`/
  `vol_sell_single_call_executor`。

## 钦定修法

- executor 落 dcd executor 同构（同一 quote→trade→status 三环），
  构型差异=腿集（LockedCcyCall+LockedUsdtPut vs 单腿）。
- paper 径先行（宪章 paper sandbox 段）——新族首评走 paper 仓
  不直开真金；模板 enabled 前 executor+paper 径必须都在。
- 顺带：surface-gap 假想卡被「无 paper 径」拦的同款门闸，executor
  落地后该卡提案自动解锁。

## 验收锚

- vol_sell eval 判词从 executor_missing 转 proposed/rejected 实评。
- paper 仓开出首笔 vol_sell 单，ledger 行落账。
