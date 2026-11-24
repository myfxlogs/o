-- fin_0248 账户名册 fin_venue_roster（工单O-37448 多账户面根修A——设计
-- SSOT /opt/o-research/designs/venue-multi-account.md D1）：
--   venue=账户身份 token（数据面 venue 键+执行路由键+凭据键前缀一身三任）；
--   proto=venue 类型（mt5|cex|dex|mt4——CFD 同质面一件适配器服务全 CFD 商，
--   CEX 异质面按所 adapter）；cred_ns=app_secrets 键前缀；roles=授权面
--   （{collect,execute}——XM 全权授权在案业主令，execute 本票空转零副作用，
--   O-37449 落地即兑现）；state=生命周期机（disabled/removed=停会话停采集，
--   历史永存）。
--   UNIQUE(proto,login) 防同户双录（cex login NULL 不撞——PG NULL 不等义）。
--   venue/proto/cred_ns/login 建后不可变——换户=新行（UPDATE 只许
--   state/last_error/updated_at/meta 域，应用层自律）。
--
-- 种子行自 app_secrets 凭据族实枚举派生（INSERT...SELECT WHERE EXISTS——
-- 凭据缺件的行如实不种，禁凭空立户）；login=出面户号（凭据本体仍在
-- app_secrets 密文列，明文户号=审计行 #35423 在案值）。
--   mt5=Exness 3001157（存量单键凭据族 mt5.*——零数据迁移语义不变）；
--   xmglobal=XM 110660304（xmglobal.mt5.* 族——验连过 eq=1000 在案；
--   roles 实授 {collect,execute}——业主令录户即全权，角色=授权记录
--   非就绪判据，方言行未落=translate fail-closed 是工程就绪非授权门槛）；
--   okx/okx_read=存量 CEX 双户（okx_read.* 凭据族在库先例接活）；
--   hyperliquid/ibkr_paper=登记册在案 venue（roles={} 只读态零消费）。

CREATE TABLE IF NOT EXISTS fin_venue_roster (
    venue      text PRIMARY KEY,
    proto      text NOT NULL,
    cred_ns    text NOT NULL,
    login      text,
    roles      text[] NOT NULL DEFAULT '{}',
    state      text NOT NULL DEFAULT 'registered'
               CHECK (state IN ('registered','connecting','active','degraded','disabled','removed')),
    last_error text,
    alias      text,
    meta       jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (proto, login)
);

COMMENT ON TABLE fin_venue_roster IS '账户名册（O-37448——venue token 一身三任：数据面 venue 键+执行路由键+凭据键前缀；增=凭据向导+upsert，减=disabled→removed 墓碑，历史永存）';

INSERT INTO fin_venue_roster(venue,proto,cred_ns,login,roles,state,alias)
SELECT v.venue, v.proto, v.cred_ns, v.login, v.roles::text[], v.state, v.alias
FROM (VALUES
    ('mt5',         'mt5',      'mt5',          '3001157',  '{collect,execute}', 'active',      'Exness',      'mt5.'),
    ('xmglobal',    'mt5',      'xmglobal.mt5', '110660304', '{collect,execute}', 'registered',  'XM',          'xmglobal.mt5.'),
    ('okx',         'cex',      'okx',          NULL,        '{collect,execute}', 'active',      'OKX',         'okx.'),
    ('okx_read',    'cex',      'okx_read',     NULL,        '{}',               'registered',  'OKX只读',     'okx_read.'),
    ('hyperliquid', 'dex',      'hyperliquid',  NULL,        '{}',               'registered',  'Hyperliquid', 'hyperliquid.'),
    ('ibkr_paper',  'cex',      'ibkr.paper',   NULL,        '{}',               'registered',  'IBKR纸盘',    'ibkr.paper')
) AS v(venue,proto,cred_ns,login,roles,state,alias,match)
WHERE EXISTS (SELECT 1 FROM app_secrets s WHERE s.key LIKE v.match || '%')
ON CONFLICT (venue) DO NOTHING;
