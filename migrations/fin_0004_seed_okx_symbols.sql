-- 0004 标的集播种入 app_config（业务配置单源=DB；USDT 计价=业主令杂币不采）。
-- 热启：collector 按代次重载换订阅面——改行即生效，不走发版。
INSERT INTO app_config(key, value)
VALUES ('okx.symbols', '["BTC-USDT-SWAP","ETH-USDT-SWAP","BTC-USDT","ETH-USDT"]')
ON CONFLICT (key) DO NOTHING;
