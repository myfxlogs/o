# XMGlobal 218 符全量实拉 + 策略可用面对拍（O-41374）

> 采集：2026-11-26 · tools/swapsurvey（SymbolList/SymbolParams/GetQuote/RequiredMargin 只读 RPC）
> 原档：`docs/能力账/xmglobal-survey-2026-11-26.json`（217 符 specs + bid/ask + margin 实拉）
> 账户：login=110660304 · leverage=1000 · USD · equity=balance=free_margin=**$1000.00** · errs=0
> 基线对拍：`docs/能力账/mt5-swap-survey-2026-10-25.md`（exness 366 符 T-0099）

## 首行判词

**「$1000 在 XM 上能跑哪几个评估族」→ 现在一个都跑不了；符号面开门后可跑 2 个（mt5_dir_mom、grid_cfd），funding-hedge/xvenue 坐实不适用。**

阻塞点不是 XM 品种或保证金，而是**数据馈送**：`app_config` 无 `bridge.xmglobal.symbols` 键 → collector 不起 XM 报价腿 → `md_klines`/`md_ticks` venue='xmglobal' **全零行**（实测）→ 全部评估臂（宇宙=venue klines/ticks 谓词）物理上看不到 XM。下文「能/不能」均按「若开馈送」口径分层判。

## 符号全景（217 符，0 错误）

| 维度 | 分布 |
|---|---|
| trade_mode | FullAccess=177 / Disabled=38 / CloseOnly=2 |
| 品类（全量 217） | FX 77 · 指数/篮子 40 · crypto 60（USD 本位 54+跨币 6）· 商品期货 30 · 金属 10 |
| swap_type | SwapNone=64 / MarginCurrency=44 / InPoints=109 |
| swap 双边 | 32 符双零；**51 符单边为正**（集中 exotic FX：USDMXN#/USDTRY/USDTHB/GBPJPY#/USDJPY# 等） |

**与 exness 对拍**（规范化剥 `#`/`m`/`Cash`/期月）：交集 **90** 个标的；XM 独有 108（主题篮子 ChinaInternet#/FAANGs_10#/TrumpWinners#、商品期货 OIL-*/COFFE-*、crypto 山寨）；exness 独有 278（主力=美股个股 CFD AAPLm/AMDm…——**XM 无个股 CFD**）。

**crypto CFD 54 名**（`BTCUSD#`/`ETHUSD#`/`SOLUSD#`/`XRPUSD#`…）——OKX `-USDT-SWAP` 可机读映射 **48 对**（BTC/ETH/SOL/DOGE/XRP…→`*USD#`），存 `fin_venue_candidate.map_draft.pairs`。

## 三表实测（$1000 口径）

**① 真实价差**（bid/ask 实算，`spread_pts` 字段全 0=浮动价差不落字段）：
n=217，min=0.7 · med=14.4 · max=779.2 bps。FullAccess 内 spread<10bps=79 / <25bps=115 / <50bps=132 / <100bps=174。
主流/金属/指数全 <2bps：US100Cash# 0.7 · US30Cash# 0.8 · GOLD# 0.8 · GER40Cash# 0.9 · US500Cash# 0.9 · USDJPY# 1.0 · JP225Cash# 1.0 · GBPUSD# 1.1 · EURUSD# 1.2 · XAUJPY# 1.5 · UK100Cash# 1.7 · XAUEUR# 1.8；BTCUSD# 2.5 · ETHUSD# 11.1。最宽端集中 Disabled/CloseOnly 与 exotic FX/商品期货。

**② swap**（三型混存）：exness 基线全零正侧，XM **51 符单边正**——carry 原料在 exotic FX 存在（USDMXN#/USDTRY/USDTHB 等正利差货币对）；但主流对双边皆负（EURUSD# -7.83/+1.44、GBPUSD# -3.66/-3.84），crypto CFD 双边巨负（BTCUSD# -2417/-1611 InPoints、ETHUSD# -68/-46）=持仓融资吞噬。

**③ 保证金**（RequiredMargin 1 lot @lev=1000，FullAccess）：≤$50=80 符 / ≤$100=43 / ≤$500=51 / =$1000=2 / >$1000=1。**$1000 本金下 174/177 可开 ≥1 lot，绝大多数可开 2+ lot**——保证金不是约束。

## 逐族判词

| 评估族 | 判词 | 依据 |
|---|---|---|
| **mt5_dir_mom**（MT5 方向动量，宇宙=venue klines `instLike='%'`） | **不能（现状）→ 开馈送后能** | 符号面够：177 FullAccess 覆盖 FX/指数/金属/crypto CFD；保证金全过；spread 主流 <2bps。但 `internal/research/dirmom/run.go` 宇宙=md_klines venue 谓词，XM klines=0 行；代码注释自承「xmglobal 另票」。开 `bridge.xmglobal.symbols` 馈送即获 ~177 符宇宙。 |
| **grid_cfd**（CFD 网格影子族，宇宙=exness md_klines+md_ticks spread 实测） | **不能（现状）→ 条件可行** | 成本闸=spread<step/2：FullAccess 内 115 符 <25bps（step=0.5% 档）——XM spread 结构对网格**比预期好**（主流指数/金属 <2bps）。但双重缺：①eval 臂谓词硬编码 `venue='exness'`（O-41369 刻意不扩 xmglobal）；②无 klines/ticks 馈送。开馈送+谓词扩 venue 后可评。 |
| **funding-hedge / xvenue-carry** | **不能（坐实预期零）** | 族本质=OKX USDT-SWAP funding 腿+对冲腿；XM 只供 CFD 对冲腿（48 crypto 映射在案），**XM 侧无 funding 费率**（CFD swap 双边皆负=负 carry 非收益源）；族本身 paper_only（ShapeXvenue 结构性拦）。48 对映射价值=未来真要跨 venue 对冲腿时的底稿，非本族解锁键。 |
| **mt5_swap_carry**（grammar 族，swap 收割） | **原料存在·未接线** | XM 51 符单边正 swap 是 exness 基线没有的差异点——exotic FX 正利差是真 carry 原料；但族 execfam.Has=false（grammar-only paper）、无馈送、无评估臂。记档待族立项时复核。 |
| stake/earn/理财面 | 不适用 | XM CFD 户无 OKX Earn/质押对应面。 |

## venue_candidate 证据补全（md 正本已写）

- `onboard.probe.symbol_params`：217 符全量规格实载（digits/spread_pts/tick_size/tick_value/contract_size/margin_ccy/trade_mode/swap_*/margin_1lot/bid/ask）
- `symbols_total`：218→**217**（SymbolList 实拉数，spec 预期 218 差 1=目录声明与可列数口径差，非漏拉——errs=0）
- `map_draft.pairs`：48 对 OKX-USDT-SWAP→XM `*USD#` crypto CFD 映射
- `map_draft.unmatched`：按类分组计数（crypto_perp_no_cfd=491 / spot_pair=504 / misc=6），非千行平铺
- `onboard.survey_ref`：指本档 JSON+MD

## state 判词建议（报告件，不直改）

**维持 `probed`，不升 `mapped`。** 理由：mapped 语义=「符号谱够支撑任一评估族跑真」——XM 符号谱本身达标（177 FullAccess、保证金全过、spread 主流 <2bps），但**跑真三件套缺二**：①无数据馈送（md_klines/ticks venue='xmglobal' 全零，`bridge.xmglobal.symbols` 键缺）；②评估臂宇宙谓词均钉 exness/OKX（dirmom 注释「xmglobal 另票」、gridcfd 谓词 `venue='exness'`）。在馈送+谓词两缺口补齐前升 mapped=空挂。建议下一票=bridge 馈送键+白名单（~120 可交易符），落地后重评 state。

## 附录：可跑符清单（能/部分族）

**mt5_dir_mom 宇宙候选**（开馈送后，177 FullAccess 全集=宇宙上限；FA 内分布：FX 55 / 指数篮 36 / crypto 47 / 商品期货 29 / 金属 10）：
- FX 55：EURUSD#/GBPUSD#/USDJPY#/AUDUSD#/USDCAD#/USDCHF#/NZDUSD#/EURAUD#/EURGBP#/EURJPY#/USDNOK#/USDSEK#/USDDKK#/USDHKD/USDTHB/USDTRY/USDMXN#/USDCNH#/USDZAR#/USDHUF#…（含 exotic 无#尾变体，全表见 JSON `symbols[].trade_mode`）
- 金属 10：GOLD#/GOLD24-7#/GAUUSD#/XAUJPY#/XAUEUR#/XAUCNH#/GAUCNH#/SILVER#/XPDUSD#/XPTUSD#（**XAU/XAG 等价物在**：黄金 7 变体+白银 SILVER#；PLAT-JAN27=铂金期货归商品类）
- 指数/篮子 36：US100Cash#/US500Cash#/US30Cash#/US2000Cash#/US400Cash#/GER40Cash#/UK100Cash#/JP225Cash#/AUS200Cash#/HK50Cash#/EU50Cash#/FRA40Cash#/SWI20Cash#/NETH25Cash#/IT40Cash#/SpainCash#/Sing30Cash#/TaiwanCash#/CHN50Cash#/ChinaHCash#/GerMid50Cash#/CA60Cash#/USFANGCash# + 主题篮 AI_INDX#/Altcoins_INDX#/Crypto_10#/ChinaInternet#/TrumpWinners#/ElectricVehicles#/Blockchain&NFT#/Precious_Metals_Index#/LATAM_INDX# + 四个 FX_Index 篮（注：HarrisWinners#/FAANGs_10# 存在但非 FullAccess）
- crypto 47（FA）：USD 本位 41=BTCUSD#/ETHUSD#/SOLUSD#/XRPUSD#/DOGEUSD#/ADAUSD#/LINKUSD#/AVAXUSD#/DOTUSD#/LTCUSD#/BCHUSD#/ETCUSD#/XLMUSD#/ATOMUSD#/UNIUSD#/AAVEUSD#/ALGOUSD#/NEARUSD#/OPUSD#/MATICUSD#/SANDUSD#/SHIBUSD#/VAULTAUSD#/XTZUSD#/ZECUSD#…；跨币 6=BTCEUR#/BTCGBP#/BTCJPY#/ETHBTC#/ETHEUR#/ETHGBP#（Disabled 13 名 USD 本位：1INCH/EGLD/ENJ/EOS/FET/FLOW/LRC/SNX/STORJ/STX/SUSHI/UMA/ZRX）
- 商品期货 29：OIL-*/BRENT-*/NGAS-*/COFFE-*/COCOA-*/CORN-*/COTTO-*/WHEAT-*/SUGAR-*/PALL-*/HGCOP-*/SBEAN-*/GSOIL-*/PLAT-* 各到期月

**grid_cfd 候选**（FullAccess ∧ spread<25bps=115 符，spread<10bps=79 符——bid/ask 实算；主力=指数 Cash 全系+金属+主流 FX+BTCUSD#）：判据=spread_bps<step/2，step=0.5% 档阈 25bps。

**不能族附注**：funding-hedge/xvenue 零承接=XM 无 USDT 本位永续+无 funding 费率面（48 crypto CFD 映射仅对冲腿底稿）；stake/earn 系 XM 无对应面；mt5_swap_carry 原料在（51 符单边正 swap）但族未入册无臂。

## 结论与建议（不施工）

1. **解锁第一块=数据馈送**：`app_config` 增 `bridge.xmglobal.symbols`（建议先 `*USD#`+`*Cash#`+主流 FX 白名单 ~120 符，非全 217——Disabled/CloseOnly 38+2 符天然不可交易）→ collector 起腿 → md_klines/ticks 有数 → 评估臂才有宇宙。此步=配置/部署票另件。
2. **dirmom 扩 venue**：`NewRunnerVenue` 已参数化 venue（exness 先例），xmglobal 是「另票」注释预留位——馈送落地后接线成本低。
3. **grid_cfd 谓词扩 venue**：O-41369 spec 明示 xmglobal 不并入本票；XM spread 实测达标（115 符<25bps）是支持扩展的证据，留档。
4. **exness 对拍结论**：XM 增量=crypto CFD 54+主题篮子+商品期货+exotic FX 正 swap；XM 减量=零个股 CFD（exness 368 中 ~200 为美股）。两 venue 标的重叠仅 90——**非替代关系，是互补**。
5. 禁线遵守：全程只读 RPC+只读 SQL；零下单、零账户变更、零策略宇宙改动、零资金调度决策。
