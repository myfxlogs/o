-- 0057 T-0139 桥二期信号层（D-0022）：md_signal 有界状态信号表——桥=信号蒸馏
-- 车间的产物面。三族：basis_ann（季度基差年化 bps）/funding_view（当期+预测+
-- 7d 均值）/depth5（top5 深度 USD 额+spread_bps 特征值）。PK(kind,key) 同键
-- upsert 最新 ts 赢（D-0018 状态表纪律——禁时间序列化）；行数有界
-- ≈标的数×kind 数。信号=派生面非事实源——消费侧必过新鲜度闸（陈=缺席如实，
-- 禁回退行情现算冒充）；信号永不触执行径（D-0022 契约钉死）。
CREATE TABLE IF NOT EXISTS md_signal (
    kind text NOT NULL,
    key  text NOT NULL,
    val  numeric,
    meta jsonb NOT NULL DEFAULT '{}',
    ts   timestamptz NOT NULL,
    PRIMARY KEY (kind, key)
);
COMMENT ON TABLE md_signal IS
  '蒸馏信号表（T-0139/D-0022）——桥面三族特征值（basis_ann/funding_view/depth5），有界状态表同键 upsert，消费必过新鲜度闸，零执行径';
