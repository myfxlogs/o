-- 1088 T-0339 IBKR 候选卡入 scout 册（venue 幂等插行——PG-md 正本→SG 投影）。
-- kind='cfd-broker' 为四值 CHECK 内最近拟（券商面非 perp/option/dex）；
-- maker/taker_bps 记 MBT 单边全成本折 bps（佣金 0.85+exchange recovery 1.15
-- =$2.00/边 ÷ $6,334 名义 ≈3.16bps——无 maker/taker 分档，两列同值）；
-- endpoints={}：IBKR 无公开免鉴权行情端点，公开面度量臂不可达，
-- state 留 'candidate'（凭据落地后 credentialed 径另票，见 carry 枚举件）。
INSERT INTO fin_venue_candidate
    (venue,kind,endpoints,maker_bps,taker_bps,min_notional_usd,kyc_ok,api_ok,evidence,state) VALUES
    ('ibkr','cfd-broker','{}',3.16,3.16,6334,true,true,
     '{"fee_url":"https://www.interactivebrokers.com/en/pricing/commissions-futures.php",
       "margin_url":"https://www.cmegroup.com/markets/cryptocurrencies/cryptocurrency-futures",
       "interest_url":"https://www.interactivebrokers.com/en/accounts/fees/pricing-interest-rates.php",
       "playbook":"docs/venue/ibkr-凭据路书.md",
       "families":"docs/venue/ibkr-carry-族枚举.md",
       "api_compare":"docs/venue/ibkr-api-对照.md",
       "fetched":"2026-11-04",
       "note":"业主已持户=候选池唯一跳过注册瓶颈 venue；无公开端点 metrics 注定空（credentialed 另票）；现金息锚 USD 3.38%(NAV>=100k,>10k 段,BM-0.5%)；MBT 保证金≈22.6% 名义"}',
     'candidate')
ON CONFLICT (venue) DO NOTHING;
