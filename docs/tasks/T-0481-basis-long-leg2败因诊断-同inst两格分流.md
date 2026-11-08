# T-0481 quarter_basis_long leg2 开仓败因诊断——同 inst 两格一分一败（T-0480 排雷续件）

- 状态：待验（原批：CTO 立票直批 2026-11-08——T-0480 排雷续件，票钦「排雷成立与否归 DM 读面裁续」）
- 领取：无头施工席常径（swe-2-flash-high）
- 优先级：P1——T-0145 部署令前置面缺角（quarter_basis_long 凭据=fail 则该格部署令不发）
- 收口移交归：决策席
- 立项：T-0480 矩阵产跑凭据格 quarter_basis_long@okx/basis_quarter#usd=50
  verdict=fail——armed→open 格闸 fail-closed 持拦在案，败因未定性前
  该格凭据不可翻绿（禁重跑冲绿掩盖病灶）

## 实据锚（生产取证面已核）

- fin_exec_drill fail 行：`开仓执行败: leg2 败/歧义→LEGGED 回滚 leg1: state=absent`
- 升件：ops_event#4228（seat=演习 esc_class=执行径演习）
- 执行日志：SG `/tmp/simdrill-0480.log`（在案未清）——exec#1123
  `LEGGED→leg1 回滚成`，与 exec#1121/1122（perp_quarter_spread
  同 inst BTC-USD-270326 双腿成交）同拍对拍
- **分流产异**：perp_quarter_spread@okx 同 venue 同 inst（BTC-USD-270326）
  双腿全绿；basis_long 同 inst leg2 独败——方向/shape 异致歧，
  非 venue 能力缺

## 诊断域

1. leg2 败/歧义判据落点：execution 双腿序（clOrdId 定态/歧态势/回滚径）
   在 basis_long shape 下的 leg2 下单参数面——与 perp_quarter_spread
   双腿参数逐字段对拍，定位败因字段（方向/ordType/instId 方言/面值）；
2. simvenue 侧 leg2 拒因：simdrill simvenue 对 BTC-USD-270326
   卖向/合约面值的处理面是否真拒——若 sim 缺陷归 simvenue 修面
   （纸面域），若真径缺陷归 execution 修面（钱径域）；
3. `state=absent` 语义核实：leg2 订单在 simvenue 侧是否真未落
   （LEGGED 回滚正确性——回滚后零残仓是硬断言）。

## 边界

- 诊断票——产病灶定性+修法提案，实现修法归 DM 裁后续派；
- 纸面域全仿真——禁真 venue 写/禁真凭据读；
- 生产只读取证径：SG docker exec o-postgres psql + /tmp/simdrill-0480.log。

## 验收

- 败因定性判词（sim 缺陷/真径缺陷/参数方言 三取一，附代码行锚）；
- 与 perp_quarter_spread 成功格参数面对拍表；
- 修法提案（票件化——若属真径缺陷须标钱径域让 DM 按钱径道裁）；
- 禁重跑矩阵冲绿——凭据翻绿须待修法落批后重产。

## 结果回填（施工 2026-11-08）

### 败因定性判词：**sim 缺陷**（simvenue 夹具域——虚构 OKX 不存在乐器+计价币未播种）

先正票面锚误：leg2 非「BTC-USD-270326 期货腿」——fin_order_leg 实证
leg_no=1=期货卖腿（filled→rolled_back）、leg_no=2=BTC-USD 现货买腿
（failed 51008）。期货腿是 leg1，两格 leg1 完全同腿。

直接机理链（全锚）：

1. cell inst=BTC-USD-270326 → `instPair` 拆 base=BTC/quote=USD →
   `newOKXVenue` 把 `BTC-USD` 登记为 SPOT 乐器（matrix_venue.go:21-27,
   46-53）——但 OKX 实有面无此对：`docs/能力账/okx-instruments-survey-
   2026-11-06.json` SPOT 1210 行 BTC-USD* 仅 BTC-USDT/BTC-USDC；
2. 播种仅 `USDT 1e7`（matrix_venue.go:49）；`quoteCcy("BTC-USD")` 按
   instId 次段="USD"（fill.go:127-135）→ USD 余额=0；
3. leg2=buy BTC-USD TgtQuote sz=24.5784 → simvenue 拒单槽序④
   insufficient_balance → absent+51008（fill.go:75-97 afford /
   simvenue.go:141-146）——fin_order_leg#121 实证
   `venue_err_code=51008「可用余额不足——USD 需 24.553822 实余 0」`；
4. `placeAndResolve` default→`state=absent`→Execute LEGGED 回滚 leg1
   （flow.go:84-101,162-180）→exec#1123 state=rolled_back、
   ops_event#4228 升件、fin_exec_drill verdict=fail 四闸齐。

腿参数面与产线同构——**非参数方言败**：matrix `basisLegs`
（matrix_legs.go:63-96）与产线 `BuildBasisLegs`（sizing.go:215-254）
同派生律（衍生腿=-YYMMDD sell+多头腿=基键 buy TgtQuote quote 额），
leg2 参数逐字段一致（对拍表下）。唯一差异=元数据源：产线
`loadMeta("BTC-USD")`→`/public/instruments?instType=SPOT&instId=BTC-USD`
空 rows→「instruments 空」fail-closed（open.go:157-168 +
client.go:285-287 + sizing.go:242-245）——产线根本走不到下单；sim 夹具
跳过 loadMeta 直登私册，虚构乐器才让 leg2 到达 venue 拒槽。

### 与 perp_quarter_spread 成功格参数面对拍（同 inst=BTC-USD-270326，usd=50）

| 字段 | #1121 leg1（绿） | #1121 leg2（绿） | #1123 leg1 | #1123 leg2（败） |
|---|---|---|---|---|
| template | perp_quarter_spread | 同左 | quarter_basis_long | 同左 |
| venue/side | okx/sell | okx/buy | okx/sell | okx/buy |
| inst_id | BTC-USD-270326 | BTC-USD-SWAP | BTC-USD-270326 | **BTC-USD** |
| sz | 3.865 | 3.865 | 3.865 | 24.5784 |
| sz_unit | base* | contract | base* | quote |
| TgtQuote | — | — | — | true |
| Notional | 25 | 25 | 25 | 25 |
| 结果 | filled | filled | filled→rolled_back | **failed 51008** |

*sz_unit=base=datedFUT 未入 contract 域次生标错（见次生发现①）。

分流机理：「同 inst 两格分流」=多头腿派生键可交易性分野——perp 族
多头腿=基键+-SWAP（BTC-USD-SWAP 真实存在，inverse SWAP 在册）vs
basis_long 族=基键现货（BTC-USD 真实不存在）。非 venue 能力缺、非
方向/shape 异致歧。

### state=absent 语义核实（诊断域③）

- leg2 absent=**真未落**：simvenue 槽④拒单在 `v.orders[clOrdID]` 建单
  前 return（simvenue.go:143-146 拒 vs :159 建）——Resolve 必 absent
  （simvenue.go:170-173）；leg 行 state=failed+51008 留痕完整；
- 回滚正确性：rev=buy+Close=true reduceOnly（flow_rollback.go:15-28）
  →simvenue afford 槽④ Close 判过→apply pos −3.865→0；exec#1123
  state=rolled_back+fin_order_leg#122 rolled_back 标记行；
  fin_position 无 exec#1123 行（position 仅双腿成交后建
  flow.go:106-115）——**零残仓硬断言成立**；
- fail-closed 全链正确：腿落库/LEGGED 回滚/凭据 fail/升件四闸齐——
  票面「armed→open 格闸 fail-closed 持拦在案」语义验证无误。

### 真径面 corollary（verdict=fail 方向正确但机理失真——DM 裁量域）

- 产线同格同样不可执行：`BuildBasisLegs`→`loadMeta("BTC-USD")`→
  SPOT instruments 空→腿构 fail-closed（永不触下单、不触
  TerminalInstErr 码域——opener.go:139-156 open_failed 事件续试径）。
  armed→open 格闸 credential=fail 恰是**正确稳态**——该格本不可执行；
- 上游口径缺口：`evalBasis` 以 BTC-USD **index-tickers 指数价**充现货腿
  价产提案（basis.go:75-79 join `split_part(inst_id,'-',1..2)`）——
  指数≠可交易现货，评估/执行口径错位；
- 次生发现①（钱径域另立票建议）：`legCash`/`legPx` 对 -YYMMDD
  FUTURES 腿误分类——guard 只排 `-SWAP` 后缀未排 datedFUT
  （flow.go:209-229,242-256）：fin_cashflow 实证 #757/#758（exec#1123）
  及 #116 对子（exec#1121 持仓期 phantom spot_sell +24998.82 USDT/
  position_ref=37）；fin_order_leg sz_unit=base+qty_base=张数同误标。
  影响=持仓期账本现金虚增+kind 量纲语义失真（全周期净额碰巧归零）；
- 次生发现②（夹具保真度）：`newOKXVenue` 将 inverse 币本位季货
  （survey 实证 BTC-USD-270326 ctType=inverse settleCcy=BTC）建模为
  CtValCcy=USDT 线性面值——$50 演习量级不改判但口径失真。

### 修法提案（票件化）

- **件① sim 夹具对齐（纸面域，可派施工）**：`newOKXVenue` 乐器册对齐
  OKX 实有面——derived spot 仅当真实存在才登记（数据源=survey 在案
  JSON 或 /public/instruments 实拉），或 `basisLegs` 前置「基键现货
  OKX 实有面」校验 fail-closed 同产线 loadMeta 语义。必补
  BTC-USD-YYMMDD（USD 计价族）测试锚——matrix_test.go:232 现仅覆盖
  BTC-USDT-260328。修后该格 detail 应变「现货乐器不在册 BTC-USD」级
  fail——**凭据仍 fail=正确稳态，禁冲绿**；
- **件② 真径面（钱径域，DM/CTO 裁量级——本票不裁）**：
  quarter_basis_long 对 inverse 币本位季货（instFamily=BTC-USD）的
  多头腿 BTC-USD 现货在 OKX 不存在——三径裁：a) eval/提案面加
  「现货腿可交易性」核查排除；b) basis_open 派生律映射
  BTC-USD→BTC-USDT（对冲口径变更=经济案级）；c) 模板限定 USDT
  计价季货族；
- **件③ legCash/legPx datedFUT 误分类（钱径域，另立票建议）**：
  guard 扩 datedFUTRe→sz_unit=contract/qty_base=张数、cashflow 豁免。

### 边界自守

- 诊断票零代码改动——diff 仅票面回填一件；
- 生产只读取证：SG `docker exec o-postgres psql`（fin_order_leg/
  fin_execution/fin_cashflow/ops_event/fin_strategy/fin_proposal/
  md_*）+/tmp/simdrill-0480.log——零写零 venue 接触零凭据读；
- 禁重跑矩阵冲绿遵守——凭据翻绿待件①/②落批后重产（件①修后预期
  仍 fail=正确稳态）。
