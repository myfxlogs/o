-- fin_0081 TRX 交易流唯一键（T-0192 返修实证：arb 搬来 DDL 缺 tx_id UNIQUE——
-- UpsertTRXTx ON CONFLICT (tx_id) 无处落=采集臂每拍全拒（零流水根因之二）。
-- 表启用至今零行——建索引零冲突风险；arb/nox 原版此键在案（collector 幂等锚）。
CREATE UNIQUE INDEX IF NOT EXISTS fin_trx_txs_tx_id_uidx ON fin_trx_txs (tx_id);
