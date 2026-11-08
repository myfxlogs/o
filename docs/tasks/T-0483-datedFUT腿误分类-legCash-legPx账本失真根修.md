# T-0483 datedFUT 腿误分类——legCash/legPx 对 -YYMMDD FUTURES 腿判错根修（T-0481 修法件③，钱径域）

- 状态：已批（CTO 批 2026-11-08——T-0481 次生发现件③，钱径域实据在案）
- 领取：无头施工席常径（swe-2-high——钱径域升档）
- 优先级：P1（账本语义失真+产线同构——datedFUT 仓位现金行虚增）
- 收口移交归：决策席
- 立项：T-0481 诊断次生发现①——`legCash`/`legPx` 对 -YYMMDD
  FUTURES 腿误分类：guard 只排 `-SWAP` 后缀未排 datedFUT
  （flow.go:209-229,242-256）

## 产侧实据（T-0481 取证在案）

- `fin_cashflow` #757/#758（exec#1123）及 #116 对子（exec#1121
  持仓期）：phantom `spot_sell` +24998.82 USDT / position_ref=37；
- `fin_order_leg` sz_unit=base+qty_base=张数同误标；
- 影响：持仓期账本现金虚增+kind 量纲语义失真（全周期净额碰巧
  归零——非正确性救，是语义坏）。

## 根域

- `internal/trading/execution/`（flow.go datedFUTRe 扩域——guard
  正则只排 `-SWAP` 未排 `-\d{6}` datedFUT 后缀）；
- legCash/legPx 调用面逐腿 sz_unit/qty_base 判定（datedFUT 应
  =contract 域+qty_base=张数+cashflow 豁免——同 SWAP 域判据）。

## 范围

1. datedFUTRe 正则扩（`-YYMMDD` 后缀排域——与 -SWAP 同判据档）；
2. legCash/legPx datedFUT 分支：sz_unit=contract、qty_base=张数、
   cashflow 豁免（phantom spot_sell 不生）；
3. 桩测钉死断言：datedFUT 腿 cashflow 行零增+sz_unit=contract
   （修前红+修后绿双点）；
4. 存量失真行处置提案（产侧 #757/#758/#116 phantom 行——回填判
   词：保留留痕 or 校正冲销，由 DM 裁——**本票不动产侧存量**）。

## 验收

- 桩测断言绿+check_all 对口（钱径域全谱档）；
- datedFUT 双腿全生命周期 cashflow 行谱形=零 phantom spot_sell；
- 存量处置提案落结果回填（判词三选一并理由）。

## 红线

- 钱径域——实现须最小面，禁顺手重构；
- 存量 fin_cashflow 行零写（判词归 DM 裁后续）；
- 先红桩律适用（修前红断言强制）。
