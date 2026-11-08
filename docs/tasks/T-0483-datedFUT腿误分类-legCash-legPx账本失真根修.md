# T-0483 datedFUT 腿误分类——legCash/legPx 对 -YYMMDD FUTURES 腿判错根修（T-0481 修法件③，钱径域）

- 状态：待验（o-worker-t0483 收口 2026-11-08）
- 领取：无头施工席常径（swe-2-high——钱径域升档）o-worker-t0483 @ 2026-11-08 18:27 CST
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

## 结果回填（o-worker-t0483 施工 2026-11-08）

### 实现摘要

- `sizing.go` 新增 `datedFUTRe`（`-\d{6}$`——okxws/simdrill 同形）+
  `contractInst()` 合约域判据（-SWAP ∨ -YYMMDD；落 sizing.go=乐器域
  判据点，且 flow.go 行棘轮 ≤300 约束下归位）；
- `legPx` 新增 datedFUT 分支：`sz_unit='contract'` + `qty_base=fillSz`
  （票面钦定 qty_base=张数——实填张数在 contract 量纲下入账，消费锚=
  guard closeWant qtyBase>0 径；与 SWAP 域 NULL 不同形——SWAP 走
  HasSuffix 首支返 sz 张数不消费 qty_base）；
- `legCash` 豁免闸 `-SWAP`→`contractInst`：datedFUT 腿不触现径，
  phantom spot_buy/spot_sell 绝源；rollback/maker 径同函覆盖零漏。
- 桩测 `TestDatedFutLegContractDomain`：datedFUT 开（sell）+平（buy）
  双腿全生命周期——sz_unit/qty_base 断言+cashflow 零增断言
  （`inst_id ~ '-[0-9]{6}$'` join 谱）。

### 消费面闭环实证（qty_base=张数钦定的机理）

- 初版按「同 SWAP 判据」落 qty_base=NULL → `TestMatrixBasisCells`
  quarter_basis_long 格红（datedFUT 平腿 closeWant 落 (0,false)→
  SzKnown=false→want=SpotAvailBal=FUTURES CtValCcy=USDT 余额→
  减仓量错→leg1 absent→残留非零）——消费面闭包谱逮到，改回票面
  钦定 qty_base=张数后绿（master 同测 5435 绿+本支 5436 绿双证）；
- datedFUT 平腿量纲径：closeWant qtyBase>0→张数→sizedCloseLegs
  min(张数, SpotAvailBal)——生产期 SpotAvailBal 对 FUTURES inst
  返 CtValCcy(USDT) 余额（量纲不纯但 min 通常取张数=既有行为保留）。

### 先红/修后绿证据

- 修前红：`sz_unit=base qty_base=non-NULL`（张数误标币量——产线
  #757/#758 同构复现）`--- FAIL: TestDatedFutLegContractDomain`；
- 修后绿：同断言 PASS + `go test ./internal/trading/execution/` 全包
  绿（21.5s，O_TEST_PG_DSN=5436 租约）+ TestMatrixBasisCells 格绿；
- 环境红隔离实证：`TestGetCockpitHeadlessMerged`/`MissingDegrade`
  在 o-postgres-test-c:5436 红=**测试库污染非码面**——5436 public
  schema 残留整套 ops_* 表（schema 隔离破缺，拔径桩 DROP 后
  search_path 回落 public.ops_headless 致 missing 判据漂）；master
  同库同测复现同红、5435 净库过——属 T-0468 测试库卫生族，留据
  待 DM 处置（本票不碰测试库面）。

### 存量失真行处置提案（判词归 DM 裁——本票零写产侧）

**建议判词：校正冲销**（三选一中选）——

- **校正冲销**（荐）：对每条 phantom 行（fin_cashflow #757/#758/
  #116 对子）插 `recon_adjust` 等额反向行，`source_ref` 锚原腿行、
  note 指 T-0483——原行保留不删（append-only 账律），recon_adjust
  kind 即钦定对账校准径（fin_0075）。理由：#116/exec#1121 持仓期
  phantom **当前仍活**——账本现金虚增是进行时非已平账；已闭仓对子
  净额虽归零，kind=spot_sell 贴合约腿对任何按 kind 聚合的消费面
  仍是语义假行。冲销=唯一同时治理「活着的虚增+死掉的假标签」的径。
- 保留留痕（弃）：净额归零靠碰巧非语义真；存活 phantom 继续虚增。
- fin_order_leg 误标列（sz_unit='base'+qty_base=张数）：随冲销脚本
  同窗一次性 UPDATE 校 `sz_unit='contract'`（qty_base 张数值本身=
  新判据下同形合法，只纠 unit 标——语义纠偏非现金流），并入判词
  包供 DM 一并裁。

### 次生发现（超票面另立票建议，本票不动）

- `internal/guard` 同族 datedFUT 判据缺口：`closeWant()`
  (sizing.go:20) 不识 'contract' szUnit（datedFUT 靠 qtyBase>0 径
  兜底返张数——本票 qty_base=张数钦定正因此径存活）；`sizedCloseLegs`
  (sizing.go:67)/`bareBags`(bag.go:64) `HasSuffix("-SWAP")` 未排
  `-YYMMDD`——datedFUT 平腿量纲判读走 SpotAvailBal（FUTURES 返
  CtValCcy=USDT 余额非合约仓面）/Spot 残袋域。病灶同族不同面，
  建议另立票治域判据归一（contractInst 单源化），本票按红线未碰。

### 范围外确认

- 未碰 internal/guard、simdrill、okxws（既有 datedFutRe 域自洽）；
- 未写任何产侧存量行；diff ⊆ internal/trading/execution/flow.go+
  sizing.go+execute_test.go+本票。
