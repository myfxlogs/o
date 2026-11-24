-- md_1107 venue='mt5' bar='1D' 周K伪日K清册（工单O-38639 根修配套）：
-- klines.go 原 timeFrame=16408 系误搬 MQL5 ENUM_TIMEFRAMES PERIOD_D1，
-- 但 mtapi ExMany timeFrame 域单位=分钟（O-38639 实测方言图
-- {1=M1,60=H1,240=H4,1440=D1,10080=W1}）——16408 分钟≈11.4d 被网关
-- 吸附 W1，全簿 67011 行自 2021 起 100% 周日戳周K（dow=0 实测唯一值）
-- 伪装 1D 落库。根修=timeFrame→1440（真日K）+周期间距 WARN 闸；
-- 存量伪行全删——扫拍每 4h 自回填 NumBars=200 真日棒（≈10 个月
-- 交易日），陈数据系 venue 史缓存可重拉面非独有资产。
-- 影响面：消费方本即饿停（confirm=1 前沿周级翻旗=上周日棒），
-- 清册后首拍即见真日棒——无迁移窗数据回退风险（原数据非真日K）。

DELETE FROM md_klines WHERE venue='mt5' AND bar='1D';
