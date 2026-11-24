-- 1107 mt5 日K粒度纠偏清污（工单O-38593）：采集端 PriceHistoryExMany
-- timeFrame=16408 按 MT5 原生 ENUM_TIMEFRAMES 请求，mtapi 网关实为
-- 分钟制（实测 tf=1440→24h 日棒/tf=10080→168h 周棒/tf=1→1m 棒）——
-- 误投产周棒落行全周日戳（dow=0）却署 bar='1D'。臂侧 56h 陈棒界按日
-- 粒度设计对周棒结构性永不满足=mt5_dir_mom 评估臂史上零产出静默死。
-- 采集端同 commit 改 1440 分钟制；本迁移清存量误标行——谓词
-- extract(dow)=0=周棒签名（67011 行逐行实证全中；日棒周末戳非周日 00:00
-- 不误伤，即便日棒先于迁移落库亦不削）。新日棒由采集臂 4h 拍回填
-- 200 棒/符幂等重建。okx venue 行不在清域。
DELETE FROM md_klines
WHERE venue='mt5' AND bar='1D' AND extract(dow from ts)=0;
