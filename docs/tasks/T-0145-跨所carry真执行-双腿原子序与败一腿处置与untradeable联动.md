# T-0145 跨所 carry 真执行——双腿原子序+败一腿处置+untradeable 联动

- 状态：待验（返修1 收口——docs/prompts/T-0145-返修1.md 两缺陷修复完，全谱绿戳）
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

## 返修1 结果回填（[角色:施工] devin 2026-11-08——docs/prompts/T-0145-返修1.md 两缺陷钦定面）

### 缺陷① okx 腿终局吞码（untradeable 联动死径）

| 面 | 修 |
|---|---|
| flow.go Execute leg2 | 败径 `%w` 透传（`leg2 败/歧义→LEGGED 回滚 leg1: %w`）——venue APIError 上浮至 TerminalInstErr 可判；err==nil 纯歧义分支保原文 |
| flow.go placeAndResolve | 同型漏面根修：Place 原 err 存 `perr`，Resolve=absent 兜底径 `state=%s: %w` 透传——mtexec.APIError（leg1 mt5 终局）/非 51 envelope 拒不再被吞；51 族判取内层真码（envelope code=1 壳+data[0].sCode=真因→vc 取 SCode，leg 留痕同真因） |
| okxexec.TerminalInstErr | 内层 SCode 同判（51001/51003）——/trade/order 终局拒实返 code=1 壳形，只看信封码=结构性永假 |
| cmd/core xvenueOpen | `xvenueErr` 终局分类件+`cerr` 闭包盖**全 err 出口**（非仅 Execute xerr——loadMeta/腿构/价/方言/闸拒等 Execute 前 okx/mt5 面 err 同走分类；okx→okx inst、mt5→cfd canonical 各入各册，双腿+提案退役共件）；SymbolParams `GetError` payload 拒 err 化（`mtexec.PbAPIError`）同走分类 |

### 缺陷② mt5LazyCli nil-conn panic（自愈钦定违例）

| 面 | 修 |
|---|---|
| mt5LazyCli.MT5() | 懒连败返 nil（原 NewMT5Client(nil)——调用即 nil deref panic）；Conn() 注释订正（nil conn 调用=panic 非「错误上浮」） |
| mtexec.Executor | `trading()`/`mt5Cli()`/`authed()` 取用点 nil 守→`OrderResult{State:"unknown"}+errNoConn` 上浮走歧态/定态径——Place/PlaceClose/Resolve/ResolveClose 全覆盖；Cli 契约注释钉 nil 语义 |
| cmd/core xvenueOpen/Mt5Sizer | SymbolParams/ContractPos 取用点 `mc==nil` 守 err 上浮；hedgeOpener.xcli/Mt5Sizer.C 窄面化 `xvenueSess`（桩测注假缝——dcd 注伪缝同款）；open.go 增 `xmeta` 桩测缝（nil=h.loadMeta 产线形） |

### 桩测（三件必修全落）

| 桩 | 断言 |
|---|---|
| `TestXvenueLeg2TerminalPassthrough`（execution） | 穿 Execute 端到端——leg2 fake 终局 APIError→err 链 `okxexec.TerminalInstErr` 可判+rolled_back+mt5:close:t777 回滚留痕 |
| `TestXvenueOpenE2EOkxTerminal`（cmd/core） | 穿 xvenueOpen→Execute 全真径——fin_venue_untradeable(BTC-USDT-SWAP,51001) 落行+双腿退役+提案退役三件全落（堵「handler 过线径断」缺口） |
| `TestNilConnUsePointErrNotPanic`（mtexec）+`TestMt5SizerDeadSessErrNotPanic`（cmd/core） | ensure 恒败→Place/PlaceClose/Resolve/ResolveClose/ContractPos 返 err 非 panic（recover 断言） |

### 机检

build✓ vet✓ gofmt✓ import✓ env✓ rejects✓ config-write-domain✓ migrations✓ bridge-audit✓ tolerate-report✓ ops-chain✓ go test 全谱绿（`bash scripts/check_all.sh` mode=merge-scoped 戳在案——含迁移种子 pgstore+proto 附加）。

### 备注（超票面观察，留裁）

- 返修①顺带根修三处同型漏面（51 族内层码判、SCode 终局判、SymbolParams payload err 化）——皆「终局分类死径」同病灶族，diff 仍在票面施工面邻域；okxexec/order.go 超字面施工面（返修①实现必需——信封壳形实返证）。
- venue 语义彩排前置/exec.xvenue_open=false 闸位/迁移 fin_0142/既有七锚桩测全未动（返修票面「不动」清单守）。

---

## 部署前置补件（2026-11-08 业主令件·COS 立壳）

- 状态：待 CTO 裁定——**本补件裁定未落+演习凭据未绿前，T-0145 部署令不发**（merge 在案不受影响，闸的是部署拍）。
- 业主令原话：「把已毕业的，都拿来纸面过一次？相当于检查一下毕业过的有没有BUG存在」——COS 裁粒度后定前置地位。
- 业主层已定两件（不再裁）：
  - **simvenue 全周期演习 = T-0145 部署前置闸**——open→hold→强平全周期、双腿残留=0 才算绿（pos33/34 裸袋案实证：开仓绿≠平仓绿，单位错配正是 close 段炸的，不演到 close 等于没验）。
  - **粒度=模板×venue×方向排列矩阵**，不逐策略重跑（bug 在码不在策略，同组合重跑=白测）。
- 归 CTO 裁填四件：
  - 演习矩阵具体组合与判据（哪些 venue 对/方向/面值组合必演）；
  - 凭据落点（`fin_exec_drill` verdict 行复用还是新凭据面）；
  - 存量排雷范围（现挂 13 张 armed 提案是否同矩阵先过一遍）；
  - 实现归属（立施工票还是 145 返修续件）。

### CTO 裁定（2026-11-08 无头裁——四件钦定回填，实现票 T-0461）

① **演习矩阵**（格=template×venue对×direction，面值=格内两档）——
必演格=生产实占格枚举∪xvenue-carry 双向钦定格共 6 格：

| 格 | direction/腿形 | 在产占用 |
|---|---|---|
| funding-hedge@okx | short_perp_long_spot（现货多+perp空） | live34+probe1 |
| dcd_put@okx | dcd_put（产品申购非双腿径 T-0160） | paper43+armed11 |
| quarter_basis_long@okx | basis_quarter（BuildBasisLegs 双腿） | paper1+armed1 |
| perp_quarter_spread@okx | perp_quarter_spread（spread 双腿） | paper1+armed1 |
| xvenue-carry@mt5×okx | long_perp_short_cfd（mt5卖+okx买） | T-0145 新开 |
| xvenue-carry@mt5×okx | short_perp_long_cfd（mt5买+okx卖） | T-0145 新开 |

格集机器枚举禁手编：fin_strategy(live|paper|probe|degraded)
∪fin_proposal(fin_proposed|fin_armed) template×venue×direction 去重
∪xvenue 双向钦定格——词表零占格=免演登记（新增格自动入谱）。
面值档：每格两档=格内最小在产名义+中位名义（粒度/单位错配咬
pos33/34 类）；零占用格=首单规模钦定 $50 单档。
判据：双腿族=open→hold→哨戒平→对账全径+venue 双腿残留=0+现金
对平；DCD 族=申购→到期结算→账实平+锁仓余额=0（残留定义随腿形）。

② **凭据落点**：复用 fin_exec_drill 同表+加 `cell_key` 可空列
（迁移）——格键='template@venue对/direction#面值档'机读；
strategy_ref=演习 ephemeral 策略 id；毕业闸读径零影响；部署闸
读径=格集 latest verdict=pass 全覆盖。不立新表。

③ **存量排雷**：格级非提案级——13 张 armed 分属 dcd_put/
perp_quarter_spread/basis_quarter 三格全在矩阵内，格绿即排雷毕；
armed→open 消费格凭据 latest=pass 同构毕业闸 fail-closed——
格凭据未绿格内提案 hold 不放行+计数。

④ **实现归属**：立新施工票 **T-0461**（非返修续件——返修=缺陷
修复语义，本件=演习基建新功能独立验收面：simdrill 格集扩+mt5
侧 sim+凭据列+格凭据闸）。T-0145 票体收口闭合。

**部署令序**：矩阵凭据全绿（fin_exec_drill cell_key 各行 pass）
→exec.xvenue_open 真户微额彩排（原判据不动——sim 闸不替真户
彩排）→部署令。
