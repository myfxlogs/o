-- 0043 dcd 持仓基线入册（T-0101：现存 ordId=11448262 BTC-USDT-261225-95000-C
-- LIVE 0.0207 BTC——10-21 部署早于账本基线，对拍 ledger 侧键书面归因凭据；
-- 幂等：按 venue+ccy+ordId 判空再插）。
INSERT INTO fin_risk_decision(intent,decision,reason,ts,config_gen)
SELECT '{"kind":"transfer","op":"dcd_baseline","venue":"okx","ccy":"BTC","amt":0.0207,"ordId":"11448262"}',
       'allow','transfer','2026-10-21 02:00:00+00',1
WHERE NOT EXISTS(SELECT 1 FROM fin_risk_decision
    WHERE intent->>'op'='dcd_baseline' AND intent->>'ordId'='11448262');
