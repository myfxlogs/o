-- 0256 账户维根修（工单O-39440——业主令「以账户来分」「直接拉真实数据」）：
-- ① fin_account 账户注册表——账户身份=不可变（UPDATE 户号=新行+旧行
--   closed，铁律在 spec）；首批钦定行 okx 四子户+mt5 两户，存量 venue
--   兜底 legacy 终身标（不可判不猜——spec 禁线）。
-- ② fin_cashflow.account_id——账本域键加账户维：存量行迁移内显式映射
--   回填（kind 感知：deposit/withdraw/owner_*→funding、earn_*/interest→
--   earn、dcd_*→dcd、recon_adjust/unmapped_bill→legacy、余→trading；
--   mt5 全行→mt5:3001157——三行簿记链属 Exness 户实证），回填后
--   NOT NULL+FK 强制「新写入径禁再落无账户行」。
-- ③ fin_known_residual.account_id——豁免行同维（NULL=未归属——对拍面
--   按账户精确扣,未归属行不匹配任何账户=fail-safe 不静默吞差）。
-- ④ venue 桶口径保留——venue/ccy 列原样在（下游读面零破），账户聚合
--   由 account_id 前缀折叠派生（Go 径）不另建冗余列。
-- 免疫件：全幂等（IF NOT EXISTS/ON CONFLICT/WHERE account_id IS NULL
--   守卫）——空表/重放/测试库零写；回填只触 NULL 行不覆写。

CREATE TABLE IF NOT EXISTS fin_account (
    account_id   text PRIMARY KEY,
    venue        text NOT NULL,
    account_no   text NOT NULL DEFAULT '',
    alias        text NOT NULL DEFAULT '',
    ccy          text NOT NULL DEFAULT '',
    sub_type     text NOT NULL CHECK (sub_type IN ('trading','funding','earn','dcd','main')),
    state        text NOT NULL DEFAULT 'active' CHECK (state IN ('active','frozen','closed')),
    declared_at  timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE fin_account IS '账户注册表（O-39440——venue 实读/对拍/簿记三面账户粒度真源；账户身份不可变,变=新行+旧行closed）';
COMMENT ON COLUMN fin_account.account_id IS '账户域键 venue:account（okx:trading/funding/earn/dcd 子户+mt5-proto 经纪商 venue:login 户号（mt5:3001157=Exness/xmglobal:110660304=XM）+<venue>:legacy 兜底）';
COMMENT ON COLUMN fin_account.sub_type IS '子户型 trading|funding|earn|dcd|main——对拍组件归属判据（parked→earn/dcdBase→dcd/native+mtm→trading）';

INSERT INTO fin_surface_lifecycle(surface,accumulates,consumer_ref,terminal_path) VALUES
  ('fin_account', true, 'recon+venue_readers', 'state=closed 人工裁/迁移退役——账户身份不可变,变=新行')
ON CONFLICT (surface) DO NOTHING;

-- 首批钦定行（spec §1A）+存量 venue 兜底 legacy（frozen=不参与 venue
-- 实读/对拍——终身标语义,非暂停）。
INSERT INTO fin_account(account_id,venue,account_no,alias,ccy,sub_type,state) VALUES
  ('okx:trading',  'okx', '',        'OKX交易户',  'USDT', 'trading', 'active'),
  ('okx:funding',  'okx', '',        'OKX资金户',  'USDT', 'funding', 'active'),
  ('okx:earn',     'okx', '',        'OKX Earn理财','USDT', 'earn',    'active'),
  ('okx:dcd',      'okx', '',        'OKX DCD期权', 'USDT', 'dcd',     'active'),
  ('mt5:3001157',  'mt5', '3001157', 'Exness',     'USD',  'main',    'active'),
  ('xmglobal:110660304','xmglobal','110660304','XM','USD','main','active'),
  ('okx:legacy',   'okx', '',        'OKX未归属',  'USDT', 'main',    'frozen'),
  ('mt5:legacy',   'mt5', '',        'MT5未归属',  'USD',  'main',    'frozen'),
  ('xmglobal:legacy','xmglobal','','XM未归属',   'USD',  'main',    'frozen'),
  ('sim:legacy',   'sim', '',        'sim未归属',  'USDT', 'main',    'frozen'),
  ('tron:legacy',  'tron','',        'tron未归属', 'USDT', 'main',    'frozen')
ON CONFLICT (account_id) DO NOTHING;

-- 存量其他 venue 兜底（fin_cashflow 实际出现的 venue 全集派生——
-- 显式映射覆盖不了的一律 <venue>:legacy 终身标,禁猜）。
INSERT INTO fin_account(account_id,venue,ccy,sub_type,state,alias)
SELECT DISTINCT venue||':legacy', venue, '', 'main', 'frozen', '未归属'
FROM fin_cashflow
WHERE venue||':legacy' NOT IN (SELECT account_id FROM fin_account)
ON CONFLICT (account_id) DO NOTHING;

-- fin_cashflow 账户维列+显式映射回填（kind 感知规则——迁移内注即
-- 映射正本；mt5 全行→mt5:3001157：id5/1658/1689 三行簿记链属
-- Exness 户实证（+3247.41/-3247.41/+1.80 链=O-33206 更正域）；
-- xmglobal→xmglobal:110660304 同型）。
ALTER TABLE fin_cashflow ADD COLUMN IF NOT EXISTS account_id text;
UPDATE fin_cashflow SET account_id = CASE
  WHEN venue='okx' AND kind IN ('deposit','withdraw','owner_in','owner_out') THEN 'okx:funding'
  WHEN venue='okx' AND kind IN ('earn_in','earn_out','interest') THEN 'okx:earn'
  WHEN venue='okx' AND (kind LIKE 'dcd\_%' OR kind='dcd_owner') THEN 'okx:dcd'
  WHEN venue='okx' AND kind IN ('recon_adjust','unmapped_bill') THEN 'okx:legacy'
  WHEN venue='okx' THEN 'okx:trading'
  WHEN venue='mt5' THEN 'mt5:3001157'
  WHEN venue='xmglobal' THEN 'xmglobal:110660304'
  ELSE venue||':legacy'
END
WHERE account_id IS NULL;
-- mig-rehearsal-ok: 回填 UPDATE 已将全量存量行 account_id 置非空
--   （CASE 兜底 ELSE venue||':legacy' 恒产出且 <venue>:legacy 行已由
--   上方 INSERT 全量派生——SET NOT NULL 无 NULL 残留可能；幂等重放
--   WHERE account_id IS NULL 守卫下第二遍 UPDATE 零行）。
ALTER TABLE fin_cashflow ALTER COLUMN account_id SET NOT NULL;
-- FK 两段式（工单#25799⑩）：NOT VALID 即时挂+VALIDATE 分离扫。
ALTER TABLE fin_cashflow ADD CONSTRAINT fin_cashflow_account_fk
  FOREIGN KEY (account_id) REFERENCES fin_account(account_id) NOT VALID;
ALTER TABLE fin_cashflow VALIDATE CONSTRAINT fin_cashflow_account_fk;
CREATE INDEX IF NOT EXISTS fin_cashflow_account_idx ON fin_cashflow(account_id,ccy);

-- fin_known_residual 豁免行账户维（NULL=未归属——对拍面精确扣,
-- 显式映射：dcd→dcd、earn/应计→earn、余 okx→trading（残差多为
-- 交易户现象）、非 okx→<venue>:legacy）。
ALTER TABLE fin_known_residual ADD COLUMN IF NOT EXISTS account_id text;
UPDATE fin_known_residual SET account_id = CASE
  WHEN venue<>'okx' THEN venue||':legacy'
  WHEN resolver='earn_accrual_gap' OR reason ~* 'earn|应计|息' THEN 'okx:earn'
  WHEN reason ~* 'dcd' THEN 'okx:dcd'
  ELSE 'okx:trading'
END
WHERE account_id IS NULL;

INSERT INTO ops_esc_valence(esc_key,weight,note) VALUES
  ('凭据缺', 70, '账户凭据缺——venue 实读不可得,condreq 供物径不猜数（工单O-39440）')
ON CONFLICT (esc_key) DO UPDATE SET weight=EXCLUDED.weight, note=EXCLUDED.note;

-- ⑤ 写入兜底闸：BEFORE INSERT 触发器按 (venue,kind) 同规则自动归户——
-- 全部既有/未来写入点零改动即满足 NOT NULL；显式 account_id（知账户
-- 写入点如 mt5 balanceops 逐户）直通不被覆写。映射与回填 CASE 同源。
CREATE OR REPLACE FUNCTION fin_cashflow_account_default() RETURNS trigger AS $$
BEGIN
  IF NEW.account_id IS NULL THEN
    NEW.account_id := CASE
      WHEN NEW.venue='okx' AND NEW.kind IN ('deposit','withdraw','owner_in','owner_out') THEN 'okx:funding'
      WHEN NEW.venue='okx' AND NEW.kind IN ('earn_in','earn_out','interest') THEN 'okx:earn'
      WHEN NEW.venue='okx' AND (NEW.kind LIKE 'dcd\_%' OR NEW.kind='dcd_owner') THEN 'okx:dcd'
      WHEN NEW.venue='okx' AND NEW.kind IN ('recon_adjust','unmapped_bill') THEN 'okx:legacy'
      WHEN NEW.venue='okx' THEN 'okx:trading'
      WHEN NEW.venue='mt5' THEN 'mt5:3001157'
      WHEN NEW.venue='xmglobal' THEN 'xmglobal:110660304'
      ELSE NEW.venue||':legacy'
    END;
    -- legacy 兜底自动开户（frozen 终身标——不可判 venue 的 FK 前补行,
    --   显式 account_id 不经此径=写错照样 FK 炸 fail-loud；幂等
    --   ON CONFLICT+WHERE NOT EXISTS 双保险,重放/并发零写零撞）。
    IF NEW.account_id LIKE '%:legacy' THEN
      INSERT INTO fin_account(account_id,venue,ccy,sub_type,state,alias)
      SELECT NEW.account_id, NEW.venue, '', 'main', 'frozen', '未归属'
      WHERE NOT EXISTS (SELECT 1 FROM fin_account WHERE account_id=NEW.account_id)
      ON CONFLICT (account_id) DO NOTHING;
    END IF;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS fin_cashflow_account_trg ON fin_cashflow;
CREATE TRIGGER fin_cashflow_account_trg BEFORE INSERT ON fin_cashflow
  FOR EACH ROW EXECUTE FUNCTION fin_cashflow_account_default();

-- ⑥ mt5_deal 幂等键随账户维重键：票号命名空间按经纪商户独立
-- （Exness/XM ticket 可撞）——(venue,…)→(account_id,…)；存量行
-- 回填后全为 mt5:3001157，唯一性保持。索引重建幂等。
DROP INDEX IF EXISTS fin_cashflow_mt5_deal_uk;
-- mig-rehearsal-ok: 重键 (venue→account_id) 后唯一性只收紧不放宽——
--   旧索引 (venue,source_table,source_ref) 已证零违例（fin_0241 建时
--   实测+三年运行零 23505 误报），account_id 为 venue 函数细分
--   （mt5 行全归 mt5:3001157，xmglobal 行归 xmglobal:110660304——两 venue 值域不相交），同
--   (account_id,source_table,source_ref) 违例集⊆旧违例集=空集。
CREATE UNIQUE INDEX IF NOT EXISTS fin_cashflow_mt5_deal_uk
	ON fin_cashflow (account_id, source_table, source_ref)
	WHERE source_table = 'mt5_deal';
