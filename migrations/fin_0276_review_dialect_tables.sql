-- O-41378 补账：review_rw 方言三表授权——fin_0274/0275 建表时仅授
-- research_rw，但 onboard 普查/mint 臂真连角色=review_rw（FIN_PG_DSN
-- 实证，#43077 同型病灶：建表授权角色≠臂运行角色→42501 复发）。
-- 三层缺补齐：registry 读面（mint 配对索引源）/census 读写面（普查
-- 拍写）/dialect 写面（mint 落行——fin_0273 时臂只读故仅授 SELECT，
-- 41378 mint 拍是 dialect 首个自动写者）。GRANT 幂等重放无害。
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'review_rw') THEN
        GRANT SELECT ON fin_canonical_registry TO review_rw;
        GRANT SELECT, INSERT, UPDATE ON fin_venue_symbols TO review_rw;
        GRANT INSERT, UPDATE ON fin_symbol_dialect TO review_rw;
    END IF;
END $$;
