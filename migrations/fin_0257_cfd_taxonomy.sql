-- fin_0257_cfd_taxonomy.sql — 工单O-39440① CFD 类目根修（业主令 2026-11-2x：
-- 「MT5就是错的——MT4 接入时应统归 CFD；历史标记就是错误的」）
--
-- taxonomy 三层钦定：
--   类目 proto:  cex | dex | cfd   —— MT4/MT5/cTrader 全归 cfd（协议版本
--                是平台子属性非类目；原设计 proto='mt4' 并列被否）
--   平台 platform: mt4 | mt5 | ...  —— cfd 类内连接平台，读件按 platform
--                分派（今仅 mt5 平台有客户端；mt4 户入册=platform='mt4'
--                读面自动缺席不误报）
--   venue token: 经纪商名（exness/xmglobal/fxpro…）——'mt5' 作为 Exness
--                的 venue token 是历史错误标记，本迁移改名 exness
--
-- 改名面（全部幂等 WHERE venue='mt5'/proto='mt5'）：
--   ① fin_venue_roster: +platform 列；proto 'mt5'→'cfd'；venue 'mt5'→
--      'exness'；cred_ns 'mt5'→'exness.mt5'（密钥键随迁——app_secrets
--      mt5.<f>→exness.mt5.<f>，credKeySplit 三段方言径自动正确）
--   ② fin_* 数据面 venue 改名（fin_cashflow/fin_known_residual/
--      fin_strategy/fin_venue_candidate）
--   ③ SG 侧 md_* 镜像表 venue 改名（OCI 主表在 md_1109 同改——双链
--      各落；WHERE 归零=幂等）
--   ④ app_secrets 键改名（密文原样——UPDATE key 不动 ciphertext）
-- 免疫件: 全 UPDATE 带 venue='mt5'/proto='mt5' 谓词——重放/测试库
--   （fixture venue 各异）零写；ALTER ADD COLUMN IF NOT EXISTS 幂等；
--   app_secrets 改名后旧键不存在=重放零写。
ALTER TABLE fin_venue_roster ADD COLUMN IF NOT EXISTS platform text;

UPDATE fin_venue_roster
   SET proto='cfd', platform='mt5', venue='exness', cred_ns='exness.mt5'
 WHERE venue='mt5';
UPDATE fin_venue_roster SET proto='cfd', platform='mt5'
 WHERE venue='xmglobal' AND proto='mt5';

UPDATE fin_cashflow SET venue='exness' WHERE venue='mt5';
UPDATE fin_known_residual SET venue='exness' WHERE venue='mt5';
UPDATE fin_strategy SET venue='exness' WHERE venue='mt5';
UPDATE fin_venue_candidate SET venue='exness' WHERE venue='mt5';

DO $$ DECLARE t text; BEGIN
  FOREACH t IN ARRAY ARRAY['md_klines','md_ticks','md_funding'] LOOP
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name=t) THEN
      EXECUTE format('UPDATE %I SET venue=''exness'' WHERE venue=''mt5''', t);
    END IF;
  END LOOP;
END $$;

UPDATE md_account_events SET venue='exness' WHERE venue='mt5';

UPDATE app_secrets SET key='exness.mt5.'||substr(key,5)
 WHERE key LIKE 'mt5.%';
