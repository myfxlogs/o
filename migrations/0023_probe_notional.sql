-- 0021 probe 最小张帽（T-0033：exec.probe_notional_usd 与 risk.max_notional_usd 双帽取小）。
INSERT INTO app_config(key,value) VALUES ('exec.probe_notional_usd','"50"') ON CONFLICT (key) DO NOTHING;
