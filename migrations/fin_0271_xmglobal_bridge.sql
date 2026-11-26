-- fin_0271 O-41376 XM 上桌二期——①bridge.xmglobal.symbols 种键（吸收工单
-- #40291：bridge.<venue>.symbols 枚举器驱动 OCI 桥腿 quote-only 面——
-- 集=FullAccess 177符显式 list（TradeMode_Disabled 38/CloseOnly 2 天然
-- 不可交易不订；机抽自 docs/能力账/xmglobal-survey-2026-11-26.json，
-- BridgeCollector.symbols() 无通配语义禁 '["*"]' 字面）。②depth5 键迁名
-- bridge.depth5.symbols→okx.depth5.symbols（根修非压声：bridge.*.symbols
-- 被 mt5 桥腿枚举器当 venue 吃→venue='depth5' 腿死等 session_token 日志
-- 噪声；okxws/depth5.go 同 commit 换键+dmwrite/tsv/裸Get闸同登，迁移落行
-- 与码部署同列车原子——旧键消即枚举腿绝，新键生即 depth5 面续）。
INSERT INTO app_config(key,value) VALUES
  ('bridge.xmglobal.symbols', '["OIL-JAN27","LATAM_INDX#","ETHGBP#","USDMXN#","GBPJPY#","Precious_Metals_Index#","GBPUSD#","CHZUSD#","LTCUSD#","SHIBUSD#","USDJPY#","GAUUSD#","ChinaInternet#","NEARUSD#","CHFJPY#","GOLD24-7#","AUDUSD#","UK100Cash#","AVAXUSD#","ETHBTC#","US100Cash#","COFFE-MAR27","US100-DEC26","USDCNH#","USDZAR#","USDHUF#","TrumpWinners#","XAUJPY#","COCOA-MAR27","NZDCHF#","SWI20-DEC26","APTUSD#","EURSEK#","LINKUSD#","ChinaHCash#","AUDNZD#","XPDUSD#","GBPSGD#","BTCGBP#","EURDKK#","NZDSGD#","LDOUSD#","GBPCHF#","BTCJPY#","USDNOK#","OILMn-JAN27","ATOMUSD#","APEUSD#","SOLUSD#","USDPLN#","SpainCash#","CORN-MAR27","EURSGD#","VAULTAUSD#","FRA40-DEC26","Altcoins_INDX#","DOGEUSD#","ElectricVehicles#","TaiwanCash#","UNIUSD#","JPY_FX_Index#","JP225-DEC26","CADCHF#","CADJPY#","USD_FX_Index#","GER40Cash#","ZECUSD#","XAUEUR#","NETH25Cash#","XPTUSD#","XAUCNH#","ICPUSD#","NGASCash#","COTTO-MAR27","OILCash#","IT40Cash#","FILUSD#","FRA40Cash#","GBPCAD#","AUDCAD#","NGAS-JAN27","HGCOP-MAR27","USDSEK#","USDHKD#","Blockchain&NFT#","AXSUSD#","GBP_FX_Index#","XRPUSD#","EURCHF#","DOTUSD#","EUR_FX_Index#","DASHUSD#","COMPUSD#","PALL-MAR27","ALGOUSD#","WHEAT-MAR27","SGDJPY#","VIX-DEC26","UK100-DEC26","NZDUSD#","ETHUSD#","US500-DEC26","EURGBP#","GOLD#","US30Cash#","GBPNOK#","EURNZD#","SUGAR-MAR27","CA60Cash#","GBPSEK#","NZDCAD#","EU50Cash#","PLAT-JAN27","EURUSD#","USDSGD#","SWI20Cash#","HK50Cash#","EURJPY#","AAVEUSD#","Crypto_10#","EURAUD#","EURZAR#","USDCAD#","SANDUSD#","GBPAUD#","MATICUSD#","BRENT-FEB27","XLMUSD#","CHN50Cash#","US500Cash#","GAUCNH#","USDX-DEC26","ETHEUR#","EURNOK#","EURTRY#","EURCAD#","GER40-DEC26","BRENTCash#","BTCEUR#","IMXUSD#","GerMid50Cash#","CHFSGD#","SILVER#","MANAUSD#","BATUSD#","BTGUSD#","US2000Cash#","USFANGCash#","AUDCHF#","USDCHF#","SBEAN-JAN27","US400Cash#","GBPDKK#","EURHKD#","OPUSD#","EURPLN#","GRTUSD#","BCHUSD#","AUDJPY#","CRVUSD#","GSOIL-DEC26","EURHUF#","AUS200Cash#","EU50-DEC26","USDTRY#","AI_INDX#","ADAUSD#","BTCUSD#","XTZUSD#","ARBUSD#","ETCUSD#","JP225Cash#","NZDJPY#","USDDKK#","US30-DEC26","Sing30Cash#","GBPNZD#"]'::jsonb)
ON CONFLICT (key) DO NOTHING;

UPDATE app_config SET key='okx.depth5.symbols'
WHERE key='bridge.depth5.symbols';
