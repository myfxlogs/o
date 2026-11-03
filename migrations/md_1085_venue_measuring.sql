-- 1085 T-0313 六候选 venue 升 measuring（正本 PG-md——fin_0098/md_1082 种子
-- 原件不改；adapter 接入六件齐备随升）。
UPDATE fin_venue_candidate SET state='measuring'
WHERE venue IN ('bitget','gate','kucoin-futures','mexc','dydx-v4','deribit')
  AND state='candidate';
