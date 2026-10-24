-- 0030 账本 kind 扩两枚（T-0055 对账差根查）：
--   interest=Earn 利息（savings earnings 累计值事件化入账——bills 无此面实证，
--   利息只现于 /finance/savings/balance.earnings 字段）；
--   recon_adjust=基线校准一次性行（历史缺口书面归因后真值校准——来源面
--   recon_baseline_adjust 专用约定）。
ALTER TABLE fin_cashflow DROP CONSTRAINT fin_cashflow_kind_check;
ALTER TABLE fin_cashflow ADD CONSTRAINT fin_cashflow_kind_check CHECK (kind IN
  ('fee','funding_settle','trade_pnl','rollback_cost','deposit','withdraw',
   'earn_in','earn_out','interest','recon_adjust'));
