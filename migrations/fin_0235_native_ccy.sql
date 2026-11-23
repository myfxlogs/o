-- 0235 原生币种域开闸（O-34287 业主令 2026-11-23：各币各域零换算——
-- USDT/USD 不互换算，OKX 杂币原生计数）。ccy CHECK 由 {USDT,USD} 闭集
-- 放为原生币形 ^[A-Z0-9]{2,20}$（BTC/DOGE/1000PEPE 等 venue 真报币全收；
-- 形闸防垃圾串——ledger.CcyOK 同形镜像）。存量 USDT/USD 行不动。
ALTER TABLE fin_cashflow DROP CONSTRAINT IF EXISTS fin_cashflow_ccy_check;
ALTER TABLE fin_cashflow ADD CONSTRAINT fin_cashflow_ccy_check
  CHECK (ccy ~ '^[A-Z0-9]{2,20}$') NOT VALID;
ALTER TABLE fin_cashflow VALIDATE CONSTRAINT fin_cashflow_ccy_check;
COMMENT ON COLUMN fin_cashflow.ccy IS '原生币种域（O-34287：各币各域零换算——OKX 原生币全收，MT5=USD 单币；形闸 ^[A-Z0-9]{2,20}$）';
