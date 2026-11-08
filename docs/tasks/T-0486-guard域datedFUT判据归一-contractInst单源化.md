# T-0486 guard 域 datedFUT 判据归一——contractInst 单源化（T-0483 次生发现件）

- 状态：已批（CTO 批 2026-11-08——T-0483 次生发现立票，病灶同族不同面）
- 领取：无头施工席常径（swe-2-flash-high）
- 优先级：P1（钱径平腿量纲判读面——datedFUT 减仓量错判域）
- 收口移交归：决策席
- 立项：T-0483 收口次生发现——guard 面 datedFUT 判据缺口三处：
  `closeWant()`（internal/guard/sizing.go:20 不识 'contract' szUnit，
  datedFUT 靠 qtyBase>0 径兜底）、`sizedCloseLegs()`（sizing.go:67
  `HasSuffix("-SWAP")` 未排 `-YYMMDD`→datedFUT 平腿量纲判读走
  SpotAvailBal——FUTURES 返 CtValCcy=USDT 余额非合约仓面）、
  `bareBags()`（internal/guard/bag.go:64 同 suffix 判据——datedFUT
  仓误入 spot 残袋域）

## 范围

1. `internal/guard/`（closeWant/sizedCloseLegs/bareBags 判据归一——
   引用 execution.contractInst 单源或同形判据下沉共用面，禁三处
   各造正则）；
2. datedFUT 平腿量纲径：sz_unit='contract'→张数域（T-0483 已落
   legPx 供给侧），SpotAvailBal 对 FUTURES inst 的余额语义不再
   误当减仓上限；
3. 桩测断言：datedFUT 持仓 closeWant 返张数径+bareBags 不吞
   datedFUT 仓（修前红+修后绿——先红桩律适用）；
4. 禁动 execution 侧（T-0483 已收口面）与 simdrill/okxws 既有
   datedFutRe 自洽域。

## 验收

- 桩测断言绿+check_all 对口（钱径邻域全谱档）；
- datedFUT 持仓 guard 平腿谱形=张数域正确减仓；
- 判据单源化（三处同引一函——后续合约族扩展单点改）。

## 红线

- 钱径邻域最小面——禁顺手重构；
- 先红桩律适用；
- 不动存量行。
