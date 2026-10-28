# T-0215 vol-sell 族 executor——strangle/single_call 从评估到钱径

> 状态：待验 | 领取：devin-t0215 @ 2026-10-28 | 优先级：P1（唯一「数据齐+评估通+能力在账」的新族）
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


## 结果回填

**实现**（五件）：①`grammar.go` 两族 Executor 落钦定名（假想卡 id=6/7 对拍
vol_sell_strangle_executor/vol_sell_single_call_executor——新卡不再产）；
②`grammar/volsell.go`（行数闸拆分件）evalVolSell 真评——single_call=C 单腿
dcd.Evaluate 同判据域，strangle=同 uly+expiry C×P 配对复合键「Cprod+Pprod」
（各侧取高 APR 腿，组合 APR=双腿等权名义均值），判据键复用 research.dcd.*，
coins 缝 AttachCoins（cmd/research/dcd.go 接 okxCoinsFor）；proposed→
fin_proposal 提案（direction='vol_sell' 对 CHECK 域）；③lifecycle
paperTemplates 白名单+两族（paper 先行钦定）；④paper.go 记账扩两族——
strangle 复合键拆腿各摊半名义；⑤open.go 派发：single_call 走 dcdOpen 同径，
strangle=volSellStrangle 双腿序贯买（腿1成腿2败=半构歧态势 ALERT 人工对账，
DCD 无撤销面禁补单纪律沿）。
顺带：import 白名单+grammar→dcd 边（同 research 域判据复用）。

**测试**：`TestEvalVolSellReal` 绿——single_call proposed+提案落、strangle
复合键提案取高 APR P 腿；全谱 check_all 绿（check-lines 拆分后过）。
SG 实况：md_dcd_product 8094 行在架——部署后下拍 eval 即出真判词。

**偏离**：strangle 组合 APR=双腿等权名义均值（票未给口径——腿构=双边权利金
合并收割语义取等权）；半构处置=exec 返 err+ALERT（DCD 无撤销面——人工对账
径沿用歧态势禁补单纪律）。

**遗留锚**：「paper 仓首笔 vol_sell 单+ledger 行」待合并部署后——proposed
提案→paperTemplates→paper 策略→dcdPaperAccounting fill 链已全通，首笔
依赖产品面实际过闸（非确定性时点，机制已备）。
