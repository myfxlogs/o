-- 0256 工单O-38820 venue 终态守卫（fin 链=SG 镜像侧同形落册）——
-- md_1109 正本对偶：projector 桥 upsert 的终态守卫 CASE 子查询在
-- SG 库内执行，册须同池实存；7 户 flip 幂等同形（桥未到拍前 SG
-- 侧也归位——双池收敛不等桥周期）。
-- ① 终态册 DDL 与 md_1109 逐字同形（桥表同形断言口径）。
-- ② projector_rw 补 SELECT——桥 ON CONFLICT 子查询以该角色执行
--   （链外角色 IF EXISTS 门同 fin_0210 先例）。
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

DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='projector_rw') THEN
        EXECUTE 'GRANT SELECT ON fin_venue_terminal_state TO projector_rw';
    END IF;
END $$;
