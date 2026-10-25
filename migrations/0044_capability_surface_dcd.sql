-- 0043 T-0102 探员双币赢枚举面+能力面注册表（v2 架构级——对外扩张机制）：
--   ① fin_capability_surface 机器可读能力面注册表——能力账（skills）的机器
--     投影，行=「系统知道这面存在」；探员枚举面从表读（零硬编码端点清单）。
--   ② md_dcd_product 在架产品快照（每扫一帧留痕——评估可溯源；append-only）。
--   ③ fin_opp_eval.direction 闭集扩 dcd_call/dcd_put（双币赢两模板——
--     C=锁币卖call、P=锁USDT卖put；非腿对冲族）。
--   ④ fin_opp_template +surface 列（模板适用声明——与注册表行 join 构成
--     「注册+声明=探员即枚举」扩张机制）。
--   ⑤ 判据机读配置种（热启；票面风控闸初版数值）。
--   ⑥ 测试面行（enabled=false——扩张自证锚：只证「注册即被枚举」不真探）。
--   ⑦ v3：fin_surface_gap 候选新面持久队列（差扫岗——检测层永不离线）。
--   ⑧ v4：direction 闭集再扩 gap_carry/gap_hedge（面分类器自动生候选）+
--     判据 config 种（爆面上限等）。
CREATE TABLE IF NOT EXISTS fin_capability_surface (
    id             bigserial PRIMARY KEY,
    venue          text NOT NULL CHECK (venue NOT LIKE '%:%'),
    surface        text NOT NULL,
    endpoint_family text NOT NULL,
    product_family text NOT NULL,
    rw_class       text NOT NULL CHECK (rw_class IN ('read','write')),
    cost_attrs     jsonb NOT NULL DEFAULT '{}',
    enabled        boolean NOT NULL DEFAULT true,
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (venue, surface)
);
COMMENT ON TABLE fin_capability_surface IS
  '能力面注册表（T-0102 v2）——能力账 skills 的机器投影；探员枚举面唯一真源';

CREATE TABLE IF NOT EXISTS md_dcd_product (
    ts               timestamptz NOT NULL,
    venue            text NOT NULL CHECK (venue NOT LIKE '%:%'),
    surface          text NOT NULL,
    uly              text NOT NULL,
    product_id       text NOT NULL,
    direction        text NOT NULL CHECK (direction IN ('C','P')),
    strike           numeric,
    expiry           timestamptz,
    annualized_yield numeric,
    min_size         numeric,
    max_size         numeric,
    spot             numeric,
    raw              jsonb NOT NULL DEFAULT '{}'
);
SELECT create_hypertable('md_dcd_product', 'ts', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS md_dcd_product_pid ON md_dcd_product(product_id, ts DESC);

ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_direction_check;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge'));

ALTER TABLE fin_opp_template ADD COLUMN IF NOT EXISTS surface text;

INSERT INTO fin_capability_surface(venue,surface,endpoint_family,product_family,rw_class,cost_attrs,enabled)
VALUES ('okx','dcd','/api/v5/finance/sfp/dcd','dual-investment','read',
        '{"fee":"premium_only","settle":"expiry"}',true),
       ('okx','dcd-test','/test/dcd-noop','test-face','read','{"probe":"none"}',false),
       ('mt5','catalog','mt5-symbol-catalog','symbols','read','{}',false)
ON CONFLICT (venue,surface) DO NOTHING;

INSERT INTO fin_opp_template(template,venue,surface,params)
SELECT t.template,t.venue,t.surface,t.params::jsonb FROM (VALUES
  ('dcd_call','okx','dcd','{"legs":["locked_ccy_sell_call"],"settle":"expiry"}'),
  ('dcd_put','okx','dcd','{"legs":["locked_usdt_sell_put"],"settle":"expiry"}'),
  ('dcd_call','okx','dcd-test','{"legs":["locked_ccy_sell_call"],"settle":"expiry"}')
) AS t(template,venue,surface,params)
WHERE NOT EXISTS(SELECT 1 FROM fin_opp_template o
  WHERE o.template=t.template AND o.venue=t.venue AND o.surface IS NOT DISTINCT FROM t.surface);

-- v3：候选新面持久队列（差扫岗写，分类器消费——幂等 UNIQUE(venue,surface)）。
CREATE TABLE IF NOT EXISTS fin_surface_gap (
    id         bigserial PRIMARY KEY,
    venue      text NOT NULL CHECK (venue NOT LIKE '%:%'),
    surface    text NOT NULL,
    state      text NOT NULL DEFAULT 'pending'
               CHECK (state IN ('pending','registered','rejected','autorejected','unclassified')),
    metadata   jsonb NOT NULL DEFAULT '{}',
    reason     text NOT NULL DEFAULT '',
    first_seen timestamptz NOT NULL DEFAULT now(),
    last_seen  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (venue, surface)
);
COMMENT ON TABLE fin_surface_gap IS
  '面差队列（T-0102 v3/v4）——差扫岗目录对拍注册表写行；分类器全机审裁态';

INSERT INTO app_config(key,value) VALUES
  ('research.dcd.min_dist_pct',  '8'),
  ('research.dcd.max_term_days', '30'),
  ('research.dcd.min_apr',       '0.05'),
  ('research.dcd.notional_pct',  '20'),
  ('research.gap.scan_interval_min', '360'),
  ('research.gap.gen_cap',       '3')
ON CONFLICT (key) DO NOTHING;
