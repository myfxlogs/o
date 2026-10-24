-- 0030 带宽预算键（T-0057：net.budget_kbps=SG 总带预算 KB/s——默认 5120=5Mbps；
-- 预警 80%/临界 95% 阈值硬编语义入注释）。
INSERT INTO app_config(key,value) VALUES ('net.budget_kbps','"5120"') ON CONFLICT (key) DO NOTHING;
