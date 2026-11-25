-- 1109 工单O-38820 venue 终态守卫——fin_venue_terminal_state 终态册+
-- 7 户 rejected 归位（PG-md 正本；实证病灶：fin_0254 只翻 SG 镜像，
-- projector 桥 md→sg 全量 upsert 把 state=EXCLUDED.state 刷回
-- 'recommended'——终态无守卫=不可逆语义被镜像掩埋）。
-- ① 终态册：state 值→是否终态的唯一判据源（常量册形——新增终态=
--   CHECK 扩值+本册补行，码面守卫自动覆盖不硬编码名单；banned/dead
--   预置——CHECK 扩值落地即被守卫自动覆盖）。
-- ② 归位 flip：7 户（业主令#37640 封轴集）md 正本翻 rejected——
--   幂等 WHERE 终态豁免（已终态不动=同形守卫）。
CREATE TABLE IF NOT EXISTS fin_venue_terminal_state(
    state text PRIMARY KEY
);
INSERT INTO fin_venue_terminal_state(state) VALUES
    ('rejected'),('banned'),('dead')
ON CONFLICT (state) DO NOTHING;

COMMENT ON TABLE fin_venue_terminal_state IS
  'venue 候选终态册（O-38820）——state 值终态判据唯一源；终态=不可逆，复活须手工令径';

UPDATE fin_venue_candidate SET state='rejected'
WHERE venue IN ('binance-usdm','bitget','bybit','dydx-v4','gate','kucoin-futures','mexc')
  AND state NOT IN (SELECT state FROM fin_venue_terminal_state);
