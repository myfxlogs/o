# T-0487 datedFUT 幻影现金行存量处置——recon_adjust 校正冲销+leg 标纠执行件（T-0483 存量判词钦定）

- 状态：已批（CTO 批 2026-11-08——T-0483 存量处置判词钦定=校正
  冲销；票面禁「本票不动产侧存量」，执行归另立窄面生产写件）
- 领取：无头施工席常径（swe-2-high——产侧 fin_ 写域执行件）
- 优先级：P2（语义假行+持仓期虚增治理——非止血，可排队）
- 收口移交归：决策席
- 立项：T-0483 判词钦定=校正冲销——对 phantom 行插
  `recon_adjust` 等额反向行（append-only 账律，原行保留不删），
  source_ref 锚原腿行、note 指 T-0483；同窗一次性 UPDATE
  fin_order_leg 误标 `sz_unit='contract'`（qty_base 张数值=新判据
  下同形合法，只纠 unit 标）

## 产侧对象（T-0483 取证在案——施工直查复核后执行）

- `fin_cashflow` #757/#758（exec#1123）+ #116 对子（exec#1121
  持仓期）——phantom spot_sell/spot_buy 贴 datedFUT 合约腿；
- `fin_order_leg` 同窗 datedFUT 腿 sz_unit='base' 误标行
  （qty_base=张数值不纠，仅 unit 标纠偏）。

## 钦定语义

- 冲销行：kind='recon_adjust'（fin_0075 钦定对账校准径），
  amount=-原行 amount，ccy 同原，source_ref=原腿行 id，
  note='T-0483 datedFUT phantom 冲销（原行#<id>）'；
- UPDATE 域：fin_order_leg 仅 datedFUT（`inst_id ~ '-[0-9]{6}$'`）
  且 sz_unit='base' 行→'contract'——逐行列改前快照入收口回填；
- 执行径：SG `docker exec o-postgres psql` 显式事务（先 SELECT
  对象行集直查对拍票面→单事务冲销+UPDATE→复核 SELECT 落回填）；
- ops_event 处置行同拍（seat=施工席 event=处置 note 指 T-0487+
  行数账）。

## 红线

- 禁删原行（append-only——只插反向行）；
- 禁动对象集外任何行（SELECT 清单先对拍票面清单，溢出=停手呈裁）；
- 凭据零明文落票/日志/会话；
- 禁碰 fin_cashflow/fin_order_leg 外任何表。

## 验收

- 对象行集直查表（改前快照+改后复核）落收口回填；
- 冲销后按 kind 聚合 datedFUT 腿现金净额=零 phantom 残留；
- ops_event 处置行在案。
