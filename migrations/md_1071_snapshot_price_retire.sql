-- 1071 T-0171 快照价格全面退役（价格源律——业主令 2026-10-26：拒绝任何快照借口）：
--   spotpoll REST 轮询写径（channel='spot_last'/'mark_price'）已随码面全撤，
--   存量行=僵尸（状态表无保留期永陈）——一次性清零；
--   频道方言归一：mark_price（REST 方言键）绝，mark-price（WS 原名）唯一键域
--   ——同键双行病灶一并了；
--   条款/目录注册面豁免声明钉列注释（spot 列=溯源快照非价格源）。

DELETE FROM md_mark_index WHERE channel NOT IN ('mark-price','index-tickers');

COMMENT ON COLUMN md_mark_index.channel IS
  'WS 频道原名（mark-price/index-tickers 唯一键域）——REST 轮询行已退役清零（T-0171 价格源律）';
COMMENT ON COLUMN md_dcd_product.spot IS
  '枚举拍溯源快照——非价格源（价格消费只准 WS 流径行 md_ticks，T-0171）';
