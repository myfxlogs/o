# T-0482 sim 夹具对齐——newOKXVenue 乐器册对齐 OKX 实有面（T-0481 修法件①）

- 状态：完成（决策席裁决通过 2026-11-09——判词在案·取证=收口并master·机械轨票翻）
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

## 结果回填（施工 2026-11-08）

### 落点（diff ⊆ 票面涉及面）

- **件① 实有面数据源**：新增 `internal/trading/simdrill/okx_surface.go`
  +`okx_spot_surface.txt`——survey JSON
  （`docs/能力账/okx-instruments-survey-2026-11-06.json` SPOT
  state=live）distilled 1210 对 sorted 全量，`go:embed` 冻结快照
  （migrations.go embed 先例同径），`okxSpotLive()` 在册判；
- **件② newOKXVenue 实有面闸**（票面写 simvenue/matrix_venue.go，
  实体在 simdrill/matrix_venue.go:55-58）：derived spot 仅当
  `okxSpotLive(spotID)` 在册才 `WithInstrument`——实有面无此对不登，
  腿构「现货乐器不在册」fail 机理对齐产线 loadMeta instruments 空
  rows fail-closed（BTC-USD 虚构现货不再放腿到 venue 走槽④51008）；
- **件③ basisLegs 可选径裁定=不另加**：venue 册 spot 域已被实有面
  闸收窄后，basisLegs 既有 `vs.Instrument(base)` 不在册判
  （matrix_legs.go:87）即同产线 loadMeta fail-closed 语义——前置
  重复校验零增量，票面「可选径」从简不做；
- **件④ USD 计价族覆盖锚**：matrix_test.go 新增
  `TestMatrixBasisUSDQuoteCell`——
  BTC-USD-270326（survey 实有 FUTURES 键，同 T-0481 生产实案族）
  quarter_basis_long@usd=50 格断言 verdict=fail+detail 含
  「现货乐器不在册 BTC-USD」+fin_exec_drill 凭据 verdict=fail 落账
  （正确稳态禁冲绿钉死）；
- **夹具联动必改**（件②推论非超范围）：实有面闸后 FOO/BAR/BATCH
  虚构对 derived spot 不再入册——funding 格腿构 fail-closed，
  TestMatrixFundingCellTwoTiers 换 BTC/ETH-USDT-SWAP、
  TestMatrixRunBatch 换 SOL-USDT-SWAP（全实有面对——与产线
  loadMeta 语义同向：虚构 inst 产线本就不可交易）。

### 验证

- `go build ./...`+`go vet ./...`+`check_migrations.sh` 全绿；
- `O_TEST_PG_DSN=…5434 go test ./internal/trading/simdrill/` 全绿
  （含新锚⑥b——USD 计价族 fail=正确稳态断言过）；
- check_all 首跑（租约落 5436）：全门禁绿+全谱 go test 仅三红——
  cmd/core cockpit 两测+risk DcdOwnerDomainExcluded，**范围外环境
  病灶不施工**：KB `testdb-residue-deterministic-red`（2026-11-08
  在案）实证=5436 库残留确定性红、5434/5435/5437/5438 四库同测全绿
  （本席复验同结论）——择净库 5438 复跑 check_all 绿戳对口；
  库清残根治归环境治理另票（KB 已载立票面建议）；
- .txt 数据件触发 check_all full 档全谱（非 scoped）——已跑绿。
