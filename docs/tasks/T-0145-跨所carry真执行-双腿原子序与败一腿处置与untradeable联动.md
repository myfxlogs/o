# T-0145 跨所 carry 真执行——双腿原子序+败一腿处置+untradeable 联动

- 状态：已批（业主令 2026-11-07「嗯，交给DM去处理」+「踢DM起来，立派验收部署」）
- 归口：钱径（真钱 venue 写面——DM 裁域）
- 领取：内联（执行=人工径——DM 裁域件，depgate/机器施工禁派）
- 号源：T-0145——T-0144/T-0330 引用在档的「另票」真身

## 背景锚（取证面，裁词留白归 DM）

- T-0144 已毕：xvenue_carry 评估+paper 全链通，armed→open 闸对模板
  显式永拒（opener.go:74——「执行原子性语义未建，抢跑=裸腿风险」）。
- 实证：armed_ttl_expired 淤积——11-07 单日 45 件退役（42 BTC 域），
  carry 提案有 armed 无出路。
- 钦定范围（T-0144 注记在案）：两腿序+败一腿处置+venue_untradeable
  联动。
- 业主红线：真钱 venue 写=人工径；首单 venue-语义彩排在先；
  首单规模/开闸判据/票面钦定全归 DM 裁定回填。

> 本票=DM 裁域空票——范围/判据/验收由 DM 裁定回填成文。

## DM 裁定（2026-11-07 决策席内联裁——判词回填，业主授权面内定案）

### 钦定范围

1. **双腿原子序**：xvenue-carry 主行（venue='okx'）live→open 驱动
   「mt5 cfd 腿先行→okx perp 腿」原子序；副腿行（venue='mt5'）永不
   独立开仓（openLive 跳过+计数）。腿序钦定 leg1=mt5 cfd（受限 venue
   先证可开——方言/TradeMode/min-lot 预检面先过，回滚成本=
   OrderClose(ticket) 已在案实证）；leg2=okx perp。
   向位映射：long_perp_short_cfd→mt5 sell+okx buy；
   short_perp_long_cfd→mt5 buy+okx sell；他向位 fail-closed 拒。
2. **败一腿处置**：复用 execution.Execute LEGGED→rollback 原子件——
   leg2 败/歧义→exec 'legged'→回滚 leg1（mt5 腿回滚=OrderClose 携
   ticket——fin_order_leg.venue_ticket 新列落仓凭据）；回滚败=
   rollback_failed+告警；任一腿歧态势=告警升件人查（歧义=人查非静默，
   禁补单同 okx 纪律）。mtexec 歧义码集修——REQUOTE*/PRICE_CHANGED/
   REQUEST_*/TIMEOUT/SERVER_BUSY/NO_PRICES 等瞬态族→unknown 走
   Resolve（OpenedOrders+OrderHistory 双面查——开腿仓在 OpenedOrders，
   历史面查不到=假 absent 病灶根修），禁一码躺 absent 误报未成交。
3. **untradeable 联动**：开仓前双 inst 黑名单预检（okx inst+cfd
   canonical——命中即退双腿+提案，零 venue 调用）；okx 腿
   TerminalInstErr 沿用现径+双腿+提案退役；mt5 腿终局码
   （SYMBOL_NOT_FOUND/INVALID_SYMBOL/TRADE_DISABLED）→mtexec.APIError
   终局位→fin_venue_untradeable(cfd canonical——0062 trigger 自动退役
   mt5 副行）+主行退役+主提案退役。MARKET_CLOSED/REQUEST_ERROR/
   REQUOTE 族=瞬态非终局，禁入册（市闭=会话复开非永拒）。
4. **平仓径修**：legSet.reverse per-leg venue（父腿 venue 抄送双腿=
   跨 venue 病灶根修）+CloseLeg.Ticket 透传 venue_ticket（hedge 户
   OrderClose 必携——无票=fail-closed 禁反向单）+Warden sizer 按 venue
   路由（mt5 腿=Mt5Sizer——OpenedOrders 实读 lots 现量；venue 无路由=
   fail 不开）+sz_unit 枚举扩 'lot'（量纲=手数对齐 venue 语义）
   +残袋判读 mt5 腿走合约残仓径（CFD 残仓=敞口事故升件非资产挂账）。
5. **装配**：execR 双腿路由 RouterExecutor{okx,mt5}——mt5 腿会话=
   懒连会话件（首用 OpenSession；RPC 败标 suspect 下次重连——常驻
   会话跨重启/断链自愈）；open/close 两径同路由同会话件。
6. **暗发闸**：`exec.xvenue_open` 默认 false（钱径新径默认关=灰度铁律
   ——venue 语义彩排过后 dmconfig 翻 true 开闸，ops_event 参数调整行
   溯源本票）。

### 首单规模

min(exec.probe_notional_usd 在档 $50, exec.max_notional_usd 在档 $5000)
=$50 双帽取小沿用；cfd min-lot 构不出（腿额<min_lot×ct_val×px）=
fail-closed 拒单（事件续试——规模不达标不裸放，禁降阈凑单）。

### 开闸判据

live_switch=true（在档）+ probe→live 毕业链沿用（paper 三判据+
execDrillPass——drill 对 okx 面机制有效，mt5 面由 venue 语义彩排
补证）+ lifetimeOK 方向修正（xvenue 向位前缀判：long_perp_* 收负租
径——rate>flip_bps 拒；short_perp_* 收正租径——rate<-flip_bps 拒，
funding 未反题才开）。breakevenOK 不适用 xvenue（成本模型=okx 双腿
专用——上游 eval 已 funding+swap+spread+slippage 四件分解过阈，
重复闸=双计成本误杀）。

### 验收锚

- 桩测谱（check_all 内）：①双腿序断言——fake 路由器证 mt5 腿先于
  okx 腿；②leg2 败→mt5 腿 OrderClose(ticket) 回滚断言+state=legged/
  rolled_back+venue_ticket 落列；③腿歧义→告警断言；④mt5 终局码→
  cfd 黑名单+双腿退役+提案退役三件全落；⑤副腿行 openLive 跳过+
  lifetimeOK 双向位符号断言；⑥guard——reverse per-leg venue+ticket
  透传+sizedCloseLegs mt5 路由+残袋路由；⑦mtexec——Resolve
  OpenedOrders 面+GetLots 量纲+歧义码集+APIError 终局位。
- venue-语义彩排（首单前置，业主钦定）：openfirst -structure cfd_perp
  真户微额双径实证（dialect 译名+lots 量纲+ticket 回写+OrderClose
  全周期）——彩排成=dmconfig 翻 exec.xvenue_open=true。
- 部署后实证：armed→open 放行日志（xvenue_open_forbidden 绝迹+open
  成/拒如实计数）；fin_order_leg.venue_ticket 落列非空；fin_position
  主行仓+warden legs 面 venue 分流在案。

### 施工面（涉及文件集）

migrations/fin_0142_xvenue_exec.sql；internal/store/pgstore/migrate_table.go；
internal/trading/mtexec/mtexec.go；internal/trading/execution/
{execute,flow,sizing}.go；internal/guard/{facts,guard,sizing,bag}.go；
internal/trading/execwiring/execwiring.go；
internal/portfolio/lifecycle/opener.go；cmd/core/{main,open}.go+
新 xvenue.go；各面 _test.go。
