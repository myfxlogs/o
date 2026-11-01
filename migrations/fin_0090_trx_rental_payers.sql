-- 0090 T-0265 租金分类回归根修：trx.rental_payers 键域播种——装配点热读
-- 的付款方地址集（逗号分隔字符串形）。名单=nox 生产锁定值对拍
-- （TBtvnRB——业主定性租金付款方，SG fin_trx_txs 84 笔 rental_income
-- 同址实证）。ON CONFLICT DO NOTHING——ops 写库改值不回写。
INSERT INTO app_config(key,value) VALUES
  ('trx.rental_payers', '"TBtvnRBgrVE9zZWkWfPvDSiuo65es55555"')
ON CONFLICT (key) DO NOTHING;
