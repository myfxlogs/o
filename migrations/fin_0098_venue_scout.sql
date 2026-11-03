-- 0098 T-0300 venue scout 骨架（D-0043——机器度量公开面、达阈发条件请求）。
-- ①fin_venue_candidate 候选 venue 台账（state: candidate|measuring|recommended|
--   registered|rejected）首批 seed——公开行情面无凭据 venue；CFD 经纪商二期。
-- ②scout 判据键 seed。
CREATE TABLE IF NOT EXISTS fin_venue_candidate (
    id              bigserial PRIMARY KEY,
    ts              timestamptz NOT NULL DEFAULT now(),
    venue           text NOT NULL UNIQUE CHECK (venue NOT LIKE '%:%'),
    kind            text NOT NULL CHECK (kind IN ('cex-perp','cex-option','dex-perp','cfd-broker')),
    endpoints       jsonb NOT NULL DEFAULT '{}', -- funding/mark/index/instruments 公开端点
    maker_bps       numeric,
    taker_bps       numeric,
    min_notional_usd numeric,
    kyc_ok          boolean NOT NULL DEFAULT true, -- 业主地区 KYC 可及（false=禁 recommended）
    api_ok          boolean NOT NULL DEFAULT true, -- 官方 REST+WS 交易可及
    withdraw_limit  text NOT NULL DEFAULT '',
    metrics         jsonb NOT NULL DEFAULT '{}', -- 度量榜行（scout 日拍写）
    evidence        jsonb NOT NULL DEFAULT '{}', -- 费率页 URL+抓取日期+研究员注册清单
    state           text NOT NULL DEFAULT 'candidate'
                    CHECK (state IN ('candidate','measuring','recommended','registered','rejected')),
    last_request_at timestamptz -- 条件请求重发闸（30d 内不重发）
);
COMMENT ON TABLE fin_venue_candidate IS
  'venue scout 候选所台账（T-0300）——公开面度量驱动注册决策；凭据/下单域不入';

INSERT INTO fin_venue_candidate(venue,kind,endpoints,maker_bps,taker_bps,kyc_ok,api_ok,evidence,state) VALUES
  ('binance-usdm','cex-perp','{"funding":"https://fapi.binance.com/fapi/v1/premiumIndex","instruments":"https://fapi.binance.com/fapi/v1/exchangeInfo"}',1,4,true,true,'{"fee_url":"https://www.binance.com/en/fee/futureFee","fetched":"2026-11-03"}','measuring'),
  ('bybit','cex-perp','{"funding":"https://api.bybit.com/v5/market/tickers?category=linear","instruments":"https://api.bybit.com/v5/market/instruments-info?category=linear"}',2,5.5,true,true,'{"fee_url":"https://www.bybit.com/en/fee/","fetched":"2026-11-03"}','measuring'),
  ('bitget','cex-perp','{"funding":"https://api.bitget.com/api/v2/mix/market/tickers?productType=USDT-FUTURES"}',2,6,true,true,'{"fee_url":"https://www.bitget.com/fee","fetched":"2026-11-03"}','candidate'),
  ('gate','cex-perp','{"funding":"https://api.gateio.ws/api/v4/futures/usdt/tickers"}',1.5,5,true,true,'{"fee_url":"https://www.gate.io/fee","fetched":"2026-11-03"}','candidate'),
  ('kucoin-futures','cex-perp','{"funding":"https://api-futures.kucoin.com/api/v1/ticker?symbol=XBTUSDTM"}',2,6,true,true,'{"fee_url":"https://www.kucoin.com/fee","fetched":"2026-11-03"}','candidate'),
  ('mexc','cex-perp','{"funding":"https://contract.mexc.com/api/v1/contract/ticker"}',0,5,true,true,'{"fee_url":"https://www.mexc.com/fee","fetched":"2026-11-03"}','candidate'),
  ('hyperliquid','dex-perp','{"funding":"https://api.hyperliquid.xyz/info"}',1.5,4.5,true,true,'{"fee_url":"https://hyperliquid.gitbook.io/hyperliquid-docs/trading/fees","fetched":"2026-11-03"}','measuring'),
  ('dydx-v4','dex-perp','{"funding":"https://indexer.dydx.trade/v4/perpetualMarkets"}',0.5,5,true,true,'{"fee_url":"https://docs.dydx.exchange/concepts-trading/rewards_fees_and_parameters","fetched":"2026-11-03"}','candidate'),
  ('deribit','cex-option','{"funding":"https://www.deribit.com/api/v2/public/get_funding_rate_history","instruments":"https://www.deribit.com/api/v2/public/get_instruments"}',0,5,true,true,'{"fee_url":"https://www.deribit.com/pages/information/fees","fetched":"2026-11-03"}','candidate'),
  ('okx','cex-perp','{"funding":"https://www.okx.com/api/v5/public/funding-rate","instruments":"https://www.okx.com/api/v5/public/instruments"}',2,5,true,true,'{"fee_url":"https://www.okx.com/fees","fetched":"2026-11-03","note":"基准行——同族对照源"}','measuring')
ON CONFLICT (venue) DO NOTHING; -- T-0301：升桥后 md_1082 种子已落同键行（两链并集态幂等）

INSERT INTO app_config(key,value) VALUES
  ('scout.recommend_margin_apr','0.05'),
  ('scout.persist_ratio','0.6'),
  ('scout.top_n','5'),
  ('scout.measure_interval_h','24')
ON CONFLICT (key) DO NOTHING;

-- T-0302：桥镜像表对投影写者授权（同 fin_opp_eval 权集——漏授则 venue_candidate
-- 投影每拍 permission denied 静默败，生产实证）。GRANT 幂等。
-- 写方角色授权（projector_rw=SG 镜像写方；测试库/fresh DB 无该角色时零失败——
-- T-0303 修裸 GRANT 坑：生产 SG 角色在案，测试库无角色 fail 是 0071 同款坑）。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'projector_rw') THEN
    GRANT SELECT,INSERT,UPDATE,DELETE ON fin_venue_candidate TO projector_rw;
  END IF;
END $$;
