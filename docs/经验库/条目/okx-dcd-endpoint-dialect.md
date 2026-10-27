# okx-dcd-endpoint-dialect（T-0207 实证入坑）

- quote：POST `/api/v5/finance/sfp/dcd/quote` 体 `{productId,notionalSz,notionalCcy}`
  ——GET+`sz` 参名=405 实证（notionalCcy=产品锁定币 BTC/ETH 非 USDT——
  md_dcd_product.raw 回填勿猜）。
- trade：POST `/trade` 体唯 `{quoteId}`——报价绑单，**无 clOrdId 幂等键**
  （歧态势超时/5xx 成否未知→禁补单，先 order-status/order-history 反查真态）。
- order-status：GET `?ordId=`（读面同族实证过）。
- redeem 面：redeem-quote/redeem POST `{ordId}/{ordId,quoteId}`——提前赎回
  径缺口（到期自动结算兜底；销号另票）。
