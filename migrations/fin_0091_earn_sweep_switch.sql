-- 0091 Earn 自动停泊开关（FEAT-01）：app_config 种子 earn.sweep_enabled=true
-- ——默认开=部署不改行为，切了才变；闸位=Worker.scan() sweepIdle/sweepResidual
-- 前置（pollRate/RedeemToTrading 不闸）。读败=fail-closed 停扫（earn 包纪律）。
INSERT INTO app_config(key,value) VALUES ('earn.sweep_enabled','true');
