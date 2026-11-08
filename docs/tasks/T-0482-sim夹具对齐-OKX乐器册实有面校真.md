# T-0482 sim 夹具对齐——newOKXVenue 乐器册对齐 OKX 实有面（T-0481 修法件①）

- 状态：已批（CTO 批 2026-11-08——T-0481 诊断钦定修法件①，纸面域）
- 领取：无头施工席常径（swe-2-flash-high）
- 优先级：P2（夹具保真度件——不影响真径；凭据正确稳态已靠真径 loadMeta 拦）
- 收口移交归：决策席
- 立项：T-0481 判词=sim 缺陷——matrix_venue.go `newOKXVenue` 把
  `BTC-USD` 登记为 SPOT 乐器（OKX 实有面无此对，survey JSON 在案）+
  播种仅 USDT 未播 USD——虚构乐器让 leg2 到达 venue 拒槽（51008），
  与真径 fail 机理（loadMeta 先拦「instruments 空」）失真

## 范围

1. `internal/trading/simvenue/matrix_venue.go`（夹具乐器册——derived
   spot 仅当 OKX 实有面存在才登记：数据源=/opt/o/docs/能力账/
   okx-instruments-survey-2026-11-06.json 在案快照，或
   /public/instruments 实拉谱形 fixture）；
2. `internal/trading/simdrill/matrix_legs.go` `basisLegs`（可选径——
   前置「基键现货 OKX 实有面」校验 fail-closed，同产线 loadMeta
   语义）；
3. `internal/trading/simdrill/matrix_test.go` 补 BTC-USD-YYMMDD
   （USD 计价族）覆盖锚（现仅 BTC-USDT-260328——matrix_test.go:232）；
4. 桩测断言：修后 quarter_basis_long 格 detail 应变
   「现货乐器不在册 BTC-USD」级 fail（机理同真径），**凭据仍
   fail=正确稳态禁冲绿**（票面钦定锚）。

## 验收

- 桩测断言绿+check_all 对口；
- basis_long 格失败机理从「venue 拒槽 51008」翻为「乐器不在册」
  级 fail-closed（与真径 loadMeta 语义对齐）；
- 禁改真径 execution/sizing/open 任何码位（纸面夹具域）。

## 红线

- 纸面域：零真 venue 写/零凭据读；
- 禁重跑生产矩阵冲绿（凭据重产归 DM 另派）；
- 不超范围动 simvenue 其他乐器/槽位语义。
