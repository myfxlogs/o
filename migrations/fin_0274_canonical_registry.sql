-- O-41378 件1：官名登记表 fin_canonical_registry——确认粒度「每族一行」
-- （席位每族登记一次，臂逐 venue 复用；机器只读，席位维护径=迁移种子+运维行）。
-- 别名=配对键（onboard census base_guess/suffix_guess 重组候选对拍），
-- 非模糊匹配——无别名命中=flag 不落行（同基异券防呆靠字典行数）。
CREATE TABLE IF NOT EXISTS fin_canonical_registry (
    canonical  text PRIMARY KEY,
    family     text NOT NULL,
    ct_val     double precision NOT NULL,
    quote_ccy  text NOT NULL DEFAULT 'USD',
    aliases    text[] NOT NULL DEFAULT '{}',
    enabled    boolean NOT NULL DEFAULT true,
    note       text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

-- 种子面（量刻意收小——覆盖即需）：ct_val=venue 自报表实返锚
-- （xmglobal-survey-2026-11-26.json contract_size 实拉，禁拍脑袋）。
-- crypto 族 ct_val=XM 实返 1 单位/手——OKX SWAP 0.01 微型形与 XM 1 单位
-- 标准形同名不同券不互译（XAU-USDT-SWAP=0.001oz 同律），各是各的 canonical。
INSERT INTO fin_canonical_registry(canonical,family,ct_val,quote_ccy,aliases,note) VALUES
 ('GOLD',   'metal', 100,    'USD', ARRAY['GOLD','XAU','XAUUSD'],   'XM GOLD# cs=100oz 实返；XAU*=金别名族'),
 ('GAUUSD', 'metal', 1000,   'USD', ARRAY['GAUUSD','GAU'],          'XM GAUUSD# cs=1000 实返（克金面）'),
 ('SILVER', 'metal', 5000,   'USD', ARRAY['SILVER','XAG','XAGUSD'], 'XM SILVER# cs=5000oz 实返'),
 ('BTCUSD', 'crypto',1,      'USD', ARRAY['BTCUSD','BTC'],         'XM BTCUSD# cs=1 实返；OKX SWAP=0.01 微型形不互译'),
 ('ETHUSD', 'crypto',1,      'USD', ARRAY['ETHUSD','ETH'],         'XM ETHUSD# cs=1 实返'),
 ('EURUSD', 'fx',    100000, 'USD', ARRAY['EURUSD'],               'XM EURUSD# cs=100k base 实返'),
 ('USDJPY', 'fx',    100000, 'JPY',ARRAY['USDJPY'],               'XM USDJPY# cs=100k base 实返；profit_ccy 仍美元锚定族'),
 ('GBPUSD', 'fx',    100000, 'USD', ARRAY['GBPUSD'],               'XM GBPUSD# cs=100k base 实返')
ON CONFLICT (canonical) DO NOTHING;

-- 机器只读本表（onboard/survey 臂跑 research 域角色——fin_0272 同律授面）。
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'research_rw') THEN
        GRANT SELECT ON fin_canonical_registry TO research_rw;
    END IF;
END $$;
