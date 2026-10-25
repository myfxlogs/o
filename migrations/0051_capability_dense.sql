-- 0049 能力账致密化（T-0119/D-0016 Ⅰ——skills/okx-*/mt5-*/cmc prose 逐面
-- 机读化：venue/surface/endpoint_family/product_family/rw_class/enabled/
-- evidence 指向 skill 节；cost_attrs 随面标注）。
INSERT INTO fin_capability_surface(venue,surface,endpoint_family,product_family,rw_class,cost_attrs,enabled) VALUES
-- okx 行情面（okx-market-ws 能力账——WS 公开流）
('okx','market-tickers','ws-public:tickers','quotes','read','{"src":"ws"}',true),
('okx','market-books5','ws-public:books5','quotes-depth','read','{"src":"ws","note":"能力在/采集停(D-0018 停采零读者)"}',false),
('okx','market-funding-rate','ws-public:funding-rate','perp-funding','read','{"src":"ws"}',true),
('okx','market-mark-price','ws-public:mark-price','perp-mark','read','{"src":"ws"}',true),
('okx','market-index-tickers','ws-public:index-tickers','index-quotes','read','{"src":"ws"}',true),
('okx','market-instruments','/api/v5/public/instruments','instruments','read','{"src":"rest"}',true),
-- okx 私有流（okx-private-ws 能力账）
('okx','priv-account','ws-private:account','account-frame','read','{"src":"ws"}',true),
('okx','priv-balance-and-position','ws-private:balance_and_position','position-frame','read','{"src":"ws"}',true),
-- okx 账户/资产（okx-account/okx-asset-funding）
('okx','account-balance','/api/v5/account/balance','account','read','{"src":"rest"}',true),
('okx','account-positions','/api/v5/account/positions','positions','read','{"src":"rest"}',true),
('okx','account-bills','/api/v5/account/bills','bills','read','{"src":"rest"}',true),
('okx','asset-balances','/api/v5/asset/balances','funding-acct','read','{"src":"rest"}',true),
('okx','asset-transfer','/api/v5/asset/transfer','transfer','write','{"idem":"clientId"}',true),
('okx','asset-transfer-state','/api/v5/asset/transfer-state','transfer-state','read','{"src":"rest"}',true),
-- okx Earn（okx-asset-funding savings 族）
('okx','savings-balance','/api/v5/finance/savings/balance','earn-acct','read','{"src":"rest"}',true),
('okx','savings-rate','/api/v5/finance/savings/lending-rate-summary','earn-rate','read','{"src":"rest"}',true),
('okx','savings-purchase','/api/v5/finance/savings/purchase-redempt','earn-deposit','write','{"side":"purchase"}',true),
('okx','savings-redeem','/api/v5/finance/savings/purchase-redempt','earn-redeem','write','{"side":"redempt"}',true),
-- okx 交易执行（okx-exec）
('okx','trade-order','/api/v5/trade/order','order','write','{"idem":"clOrdId"}',true),
('okx','trade-order-precheck','/api/v5/trade/order-precheck','order-precheck','read','{"src":"rest"}',true),
('okx','trade-cancel','/api/v5/trade/cancel-order','order-cancel','write','{}',true),
-- mt5 账户/行情/订单面（mt5-account/quotes-history/orders-streams）
('mt5','account','mtapi:Account','account','read','{"src":"grpc"}',true),
('mt5','account-summary','mtapi:AccountSummary','account','read','{"src":"grpc"}',true),
('mt5','symbols','mtapi:Symbols','symbol-catalog','read','{"src":"grpc"}',true),
('mt5','quote','mtapi:GetQuote','quotes','read','{"src":"grpc"}',true),
('mt5','price-history','mtapi:PriceHistory','history','read','{"src":"grpc"}',true),
('mt5','opened-orders','mtapi:OpenedOrders','positions','read','{"src":"grpc"}',true),
('mt5','order-history','mtapi:OrderHistory','orders','read','{"src":"grpc"}',true),
('mt5','stream-tick','mtapi:StreamsTick','tick-stream','read','{"src":"grpc-stream"}',true),
('mt5','required-margin','mtapi:RequiredMargin','margin','read','{"src":"grpc"}',true),
-- cmc 纯读面（T-0108 移植面）
('cmc','prices','cmc-rest:prices','quotes','read','{"src":"rest"}',true),
('cmc','account-summary','cmc-rest:account-summary','account','read','{"src":"rest"}',true),
('cmc','positions','cmc-rest:positions','positions','read','{"src":"rest"}',true),
('cmc','history','cmc-rest:history','bills','read','{"src":"rest"}',true),
('cmc','ws-stream','cmc-ws:live','live-frames','read','{"src":"ws"}',true)
ON CONFLICT (venue,surface) DO NOTHING;
