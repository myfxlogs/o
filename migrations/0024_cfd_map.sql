-- 0023 cfd_perp 跨 venue 映射（T-0036：exec.cfd_map.<inst>=<mt5 symbol>
-- ——真户 Exness m 尾方言经 SymbolParams 实读钉不猜）。
INSERT INTO app_config(key,value) VALUES ('exec.cfd_map.BTC-USDT-SWAP','"BTCUSDm"') ON CONFLICT (key) DO NOTHING;
