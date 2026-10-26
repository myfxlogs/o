-- 0021 MT5 数据面读面（T-0031——第二 venue 只读接入：行情+账户事实，零钱径）。
-- mt5.symbols=采集清单（MT5 符号方言——broker 后缀各异，值须 SymbolList 实读回填，
-- 种子空=采集岗 boot 时实读筛选 BTCUSD*/ETHUSD* 自落，禁猜名字面量）。
-- 凭据不走迁移（demo 户字样如实注）：mt5.host/login/password/server 入 app_secrets 密文列。
INSERT INTO app_config(key,value) VALUES
  ('mt5.gateway_addr', '"mt5grpc3.mtapi.io:443"'),
  ('mt5.symbols',      '[]')
ON CONFLICT (key) DO NOTHING;
