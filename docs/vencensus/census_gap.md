<!-- vencensus 差集快照（工单#25843 venue 能力面普查基线——工具产，禁手编）
     生成: 2026-11-20 10:34Z  源: okx_v5_endpoints.tsv(385) mt4.proto(49rpc) mt5.proto(74rpc) binance-code(2)
     对拍: skills/* canon + internal/cmd/tools 非 gen 码面调用点
     重跑幂等——同文件 diff 出新增/消失项 -->

## okx

### okx/account (44) —— 账户配置/仓位/费级面——已接部分，余未探

- /api/v5/account/account-level-switch-preset
- /api/v5/account/account-position-risk
- /api/v5/account/activate-feature
- /api/v5/account/activate-option
- /api/v5/account/adjust-leverage-info
- /api/v5/account/bills-history-archive
- /api/v5/account/collateral-assets
- /api/v5/account/demo-adjust-balance
- /api/v5/account/greeks
- /api/v5/account/instruments
- /api/v5/account/interest-accrued
- /api/v5/account/interest-limits
- /api/v5/account/interest-rate
- /api/v5/account/max-loan
- /api/v5/account/max-withdrawal
- /api/v5/account/mmp-config
- /api/v5/account/mmp-reset
- /api/v5/account/move-positions
- /api/v5/account/move-positions-history
- /api/v5/account/position-builder
- /api/v5/account/position-builder-graph
- /api/v5/account/position-tiers
- /api/v5/account/position/margin-balance
- /api/v5/account/positions-history
- /api/v5/account/precheck-set-delta-neutral
- /api/v5/account/risk-state
- /api/v5/account/set-account-level
- /api/v5/account/set-account-switch-precheck
- /api/v5/account/set-auto-earn
- /api/v5/account/set-auto-loan
- /api/v5/account/set-auto-repay
- /api/v5/account/set-collateral-assets
- /api/v5/account/set-fee-type
- /api/v5/account/set-greeks
- /api/v5/account/set-isolated-mode
- /api/v5/account/set-position-mode
- /api/v5/account/set-riskOffset-amt
- /api/v5/account/set-settle-currency
- /api/v5/account/set-trading-config
- /api/v5/account/spot-borrow-repay-history
- /api/v5/account/spot-manual-borrow-repay
- /api/v5/account/subaccount/balances
- /api/v5/account/subaccount/max-withdrawal
- /api/v5/account/trade-fee

### okx/affiliate (6) —— 代理返佣面（弱相关）

- /api/v5/affiliate/co-inviter/list
- /api/v5/affiliate/invitee/detail
- /api/v5/affiliate/invitee/list
- /api/v5/affiliate/link/list
- /api/v5/affiliate/performance/summary
- /api/v5/affiliate/sub-affiliate/list

### okx/asset (19) —— 资金户/划转/账单面——已接部分，余未探

- /api/v5/asset/asset-valuation
- /api/v5/asset/bills-history
- /api/v5/asset/cancel-withdrawal
- /api/v5/asset/convert/currencies
- /api/v5/asset/convert/currency-pair
- /api/v5/asset/convert/estimate-quote
- /api/v5/asset/convert/history
- /api/v5/asset/convert/trade
- /api/v5/asset/deposit-address
- /api/v5/asset/deposit-withdraw-status
- /api/v5/asset/exchange-list
- /api/v5/asset/monthly-statement
- /api/v5/asset/non-tradable-assets
- /api/v5/asset/subaccount/balances
- /api/v5/asset/subaccount/bills
- /api/v5/asset/subaccount/managed-subaccount-bills
- /api/v5/asset/subaccount/transfer
- /api/v5/asset/travel-rule/get-locked
- /api/v5/asset/travel-rule/unlock

### okx/copytrading (26) —— 跟单/带单面——信号分发 venue 代管径

- /api/v5/copytrading/algo-order
- /api/v5/copytrading/amend-copy-settings
- /api/v5/copytrading/amend-profit-sharing-ratio
- /api/v5/copytrading/close-subposition
- /api/v5/copytrading/config
- /api/v5/copytrading/copy-settings
- /api/v5/copytrading/current-lead-traders
- /api/v5/copytrading/current-subpositions
- /api/v5/copytrading/first-copy-settings
- /api/v5/copytrading/instruments
- /api/v5/copytrading/profit-sharing-details
- /api/v5/copytrading/public-config
- /api/v5/copytrading/public-copy-traders
- /api/v5/copytrading/public-current-subpositions
- /api/v5/copytrading/public-lead-traders
- /api/v5/copytrading/public-pnl
- /api/v5/copytrading/public-preference-currency
- /api/v5/copytrading/public-stats
- /api/v5/copytrading/public-subpositions-history
- /api/v5/copytrading/public-weekly-pnl
- /api/v5/copytrading/set-instruments
- /api/v5/copytrading/stop-copy-trading
- /api/v5/copytrading/subpositions-history
- /api/v5/copytrading/total-profit-sharing
- /api/v5/copytrading/total-unrealized-profit-sharing
- /api/v5/copytrading/unrealized-profit-sharing-details

### okx/fiat (5) —— 法币出入金面（红线外）

- /api/v5/fiat/buy-sell/currencies
- /api/v5/fiat/buy-sell/currency-pair
- /api/v5/fiat/buy-sell/history
- /api/v5/fiat/buy-sell/quote
- /api/v5/fiat/buy-sell/trade

### okx/finance (49) —— 理财/借贷/链上赚币停泊面——savings 已接一脉，余族未探

- /api/v5/finance/flexible-loan/adjust-collateral
- /api/v5/finance/flexible-loan/borrow
- /api/v5/finance/flexible-loan/borrow-currencies
- /api/v5/finance/flexible-loan/collateral-assets
- /api/v5/finance/flexible-loan/emode-info
- /api/v5/finance/flexible-loan/interest-accrued
- /api/v5/finance/flexible-loan/loan-history
- /api/v5/finance/flexible-loan/loan-info
- /api/v5/finance/flexible-loan/max-collateral-redeem-amount
- /api/v5/finance/flexible-loan/max-loan
- /api/v5/finance/flexible-loan/repay
- /api/v5/finance/okusd/account
- /api/v5/finance/okusd/limits
- /api/v5/finance/okusd/rate/history
- /api/v5/finance/okusd/redeem
- /api/v5/finance/okusd/redeem/history
- /api/v5/finance/okusd/rewards/history
- /api/v5/finance/okusd/subscribe
- /api/v5/finance/okusd/subscribe/history
- /api/v5/finance/savings/lending-history
- /api/v5/finance/savings/lending-rate-history
- /api/v5/finance/sfp/dcd/products
- /api/v5/finance/sfp/dcd/redeem
- /api/v5/finance/sfp/dcd/redeem-quote
- /api/v5/finance/stable-rewards/apy-history
- /api/v5/finance/stable-rewards/balance
- /api/v5/finance/stable-rewards/product-info
- /api/v5/finance/stable-rewards/quote
- /api/v5/finance/stable-rewards/subscribe-redeem-history
- /api/v5/finance/stable-rewards/trade
- /api/v5/finance/staking-defi/cancel
- /api/v5/finance/staking-defi/eth/apy-history
- /api/v5/finance/staking-defi/eth/balance
- /api/v5/finance/staking-defi/eth/cancel-redeem
- /api/v5/finance/staking-defi/eth/product-info
- /api/v5/finance/staking-defi/eth/purchase
- /api/v5/finance/staking-defi/eth/purchase-redeem-history
- /api/v5/finance/staking-defi/eth/redeem
- /api/v5/finance/staking-defi/offers
- /api/v5/finance/staking-defi/orders-active
- /api/v5/finance/staking-defi/orders-history
- /api/v5/finance/staking-defi/purchase
- /api/v5/finance/staking-defi/redeem
- /api/v5/finance/staking-defi/sol/apy-history
- /api/v5/finance/staking-defi/sol/balance
- /api/v5/finance/staking-defi/sol/product-info
- /api/v5/finance/staking-defi/sol/purchase
- /api/v5/finance/staking-defi/sol/purchase-redeem-history
- /api/v5/finance/staking-defi/sol/redeem

### okx/market (21) —— 行情面——已接部分（books-full/option 等未探）

- /api/v5/market/block-ticker
- /api/v5/market/block-tickers
- /api/v5/market/books
- /api/v5/market/books-full
- /api/v5/market/books-rpi
- /api/v5/market/books-sbe
- /api/v5/market/call-auction-details
- /api/v5/market/candles
- /api/v5/market/exchange-rate
- /api/v5/market/history-index-candles
- /api/v5/market/history-mark-price-candles
- /api/v5/market/history-trades
- /api/v5/market/index-candles
- /api/v5/market/index-components
- /api/v5/market/mark-price-candles
- /api/v5/market/option/instrument-family-trades
- /api/v5/market/platform-24-volume
- /api/v5/market/sprd-candles
- /api/v5/market/sprd-history-candles
- /api/v5/market/sprd-ticker
- /api/v5/market/trades

### okx/public (26) —— 公开品种/标记价/资金费面——已接部分，余未探

- /api/v5/public/block-trades
- /api/v5/public/convert-contract-coin
- /api/v5/public/corporate-actions
- /api/v5/public/delivery-exercise-history
- /api/v5/public/delta-hedge-currencies
- /api/v5/public/discount-rate-interest-free-quota
- /api/v5/public/economic-calendar
- /api/v5/public/estimated-price
- /api/v5/public/estimated-settlement-info
- /api/v5/public/event-contract/events
- /api/v5/public/event-contract/markets
- /api/v5/public/event-contract/series
- /api/v5/public/funding-rate-history
- /api/v5/public/instrument-tick-bands
- /api/v5/public/insurance-fund
- /api/v5/public/interest-rate-loan-quota
- /api/v5/public/market-data-history
- /api/v5/public/mm-instrument-types
- /api/v5/public/opt-summary
- /api/v5/public/option-trades
- /api/v5/public/position-tiers
- /api/v5/public/premium-history
- /api/v5/public/price-limit
- /api/v5/public/settlement-history
- /api/v5/public/time
- /api/v5/public/underlying

### okx/rfq (19) —— 大宗询价面——多腿组合 RFQ 价优径

- /api/v5/rfq/cancel-all-after
- /api/v5/rfq/cancel-all-quotes
- /api/v5/rfq/cancel-all-rfqs
- /api/v5/rfq/cancel-batch-quotes
- /api/v5/rfq/cancel-batch-rfqs
- /api/v5/rfq/cancel-quote
- /api/v5/rfq/cancel-rfq
- /api/v5/rfq/counterparties
- /api/v5/rfq/counterparties/
- /api/v5/rfq/create-quote
- /api/v5/rfq/create-rfq
- /api/v5/rfq/execute-quote
- /api/v5/rfq/maker-instrument-settings
- /api/v5/rfq/mmp-config
- /api/v5/rfq/mmp-reset
- /api/v5/rfq/public-trades
- /api/v5/rfq/quotes
- /api/v5/rfq/rfqs
- /api/v5/rfq/trades

### okx/rubik (15) —— 官方统计/多空比/期权与大数据面——ctx 源补

- /api/v5/rubik/stat/contracts/long-short-account-ratio
- /api/v5/rubik/stat/contracts/long-short-account-ratio-contract
- /api/v5/rubik/stat/contracts/long-short-account-ratio-contract-top-trader
- /api/v5/rubik/stat/contracts/long-short-position-ratio-contract-top-trader
- /api/v5/rubik/stat/contracts/open-interest-history
- /api/v5/rubik/stat/contracts/open-interest-volume
- /api/v5/rubik/stat/margin/loan-ratio
- /api/v5/rubik/stat/option/open-interest-volume
- /api/v5/rubik/stat/option/open-interest-volume-expiry
- /api/v5/rubik/stat/option/open-interest-volume-ratio
- /api/v5/rubik/stat/option/open-interest-volume-strike
- /api/v5/rubik/stat/option/taker-block-volume
- /api/v5/rubik/stat/taker-volume
- /api/v5/rubik/stat/taker-volume-contract
- /api/v5/rubik/stat/trading-data/support-coin

### okx/sprd (12) —— 价差组合原子下单面——多腿同撮防瘸腿

- /api/v5/sprd/amend-order
- /api/v5/sprd/books
- /api/v5/sprd/cancel-all-after
- /api/v5/sprd/cancel-order
- /api/v5/sprd/mass-cancel
- /api/v5/sprd/order
- /api/v5/sprd/orders-history
- /api/v5/sprd/orders-history-archive
- /api/v5/sprd/orders-pending
- /api/v5/sprd/public-trades
- /api/v5/sprd/spreads
- /api/v5/sprd/trades

### okx/support (2) —— 客服/公告面

- /api/v5/support/announcement-types
- /api/v5/support/announcements

### okx/system (1) —— 系统状态面

- /api/v5/system/status

### okx/trade (22) —— 交易单面——已接部分（algo/precheck 等未探）

- /api/v5/trade/account-rate-limit
- /api/v5/trade/amend-algos
- /api/v5/trade/amend-batch-orders
- /api/v5/trade/amend-order
- /api/v5/trade/batch-orders
- /api/v5/trade/cancel-algos
- /api/v5/trade/cancel-all-after
- /api/v5/trade/cancel-batch-orders
- /api/v5/trade/close-position
- /api/v5/trade/easy-convert
- /api/v5/trade/easy-convert-currency-list
- /api/v5/trade/easy-convert-history
- /api/v5/trade/mass-cancel
- /api/v5/trade/one-click-repay
- /api/v5/trade/one-click-repay-currency-list
- /api/v5/trade/one-click-repay-currency-list-v2
- /api/v5/trade/one-click-repay-history
- /api/v5/trade/one-click-repay-history-v2
- /api/v5/trade/one-click-repay-v2
- /api/v5/trade/order-algo
- /api/v5/trade/orders-algo-history
- /api/v5/trade/orders-algo-pending

### okx/tradingBot (63) —— venue 原生 bot 容器（网格/DCD/信号/定投/搬砖）——省自建撮合与格账（工单#25840 网格适配器在案）

- /api/v5/tradingBot/dca/amend-order-algo
- /api/v5/tradingBot/dca/create
- /api/v5/tradingBot/dca/cycle-list
- /api/v5/tradingBot/dca/history-list
- /api/v5/tradingBot/dca/margin/add
- /api/v5/tradingBot/dca/margin/reduce
- /api/v5/tradingBot/dca/ongoing-list
- /api/v5/tradingBot/dca/orders
- /api/v5/tradingBot/dca/orders/manual-buy
- /api/v5/tradingBot/dca/position-details
- /api/v5/tradingBot/dca/settings/reinvestment
- /api/v5/tradingBot/dca/settings/take-profit
- /api/v5/tradingBot/dca/stop
- /api/v5/tradingBot/grid/adjust-investment
- /api/v5/tradingBot/grid/ai-param
- /api/v5/tradingBot/grid/amend-algo-basic-param
- /api/v5/tradingBot/grid/amend-order-algo
- /api/v5/tradingBot/grid/cancel-close-order
- /api/v5/tradingBot/grid/close-position
- /api/v5/tradingBot/grid/compute-margin-balance
- /api/v5/tradingBot/grid/copy-order-algo
- /api/v5/tradingBot/grid/grid-quantity
- /api/v5/tradingBot/grid/margin-balance
- /api/v5/tradingBot/grid/min-investment
- /api/v5/tradingBot/grid/order-instant-trigger
- /api/v5/tradingBot/grid/orders-algo-details
- /api/v5/tradingBot/grid/orders-algo-history
- /api/v5/tradingBot/grid/orders-algo-pending
- /api/v5/tradingBot/grid/positions
- /api/v5/tradingBot/grid/stop-order-algo
- /api/v5/tradingBot/grid/sub-orders
- /api/v5/tradingBot/grid/withdraw-income
- /api/v5/tradingBot/public/rsi-back-testing
- /api/v5/tradingBot/recurring/add-investment
- /api/v5/tradingBot/recurring/amend-order-algo
- /api/v5/tradingBot/recurring/amend-price-range
- /api/v5/tradingBot/recurring/amend-recurring-amount
- /api/v5/tradingBot/recurring/amend-recurring-time
- /api/v5/tradingBot/recurring/order-algo
- /api/v5/tradingBot/recurring/orders-algo-details
- /api/v5/tradingBot/recurring/orders-algo-history
- /api/v5/tradingBot/recurring/orders-algo-pending
- /api/v5/tradingBot/recurring/pause
- /api/v5/tradingBot/recurring/restart
- /api/v5/tradingBot/recurring/stop-order-algo
- /api/v5/tradingBot/recurring/sub-orders
- /api/v5/tradingBot/signal/amendTPSL
- /api/v5/tradingBot/signal/cancel-sub-order
- /api/v5/tradingBot/signal/close-position
- /api/v5/tradingBot/signal/create-signal
- /api/v5/tradingBot/signal/event-history
- /api/v5/tradingBot/signal/margin-balance
- /api/v5/tradingBot/signal/order-algo
- /api/v5/tradingBot/signal/orders-algo-details
- /api/v5/tradingBot/signal/orders-algo-history
- /api/v5/tradingBot/signal/orders-algo-pending
- /api/v5/tradingBot/signal/positions
- /api/v5/tradingBot/signal/positions-history
- /api/v5/tradingBot/signal/set-instruments
- /api/v5/tradingBot/signal/signals
- /api/v5/tradingBot/signal/stop-order-algo
- /api/v5/tradingBot/signal/sub-order
- /api/v5/tradingBot/signal/sub-orders

### okx/users (12) —— 子账户族管理面

- /api/v5/users/entrust-subaccount-list
- /api/v5/users/glp/historicalperformance
- /api/v5/users/glp/todayperformance
- /api/v5/users/glp/v2/details
- /api/v5/users/glp/v2/history/performance
- /api/v5/users/glp/v2/overview
- /api/v5/users/subaccount/apikey
- /api/v5/users/subaccount/create-subaccount
- /api/v5/users/subaccount/delete-apikey
- /api/v5/users/subaccount/list
- /api/v5/users/subaccount/modify-apikey
- /api/v5/users/subaccount/set-transfer-out

## mt4

### mt4/mtapi-rpc (5) —— 未评估——研究员席判

- Groups
- QuoteHistoryMany
- RequestQuoteHistory
- SetPlacedType
- SubscribeQuoteHistory

## mt5

## binance

