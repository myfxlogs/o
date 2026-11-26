-- fin_0269 O-41364 margin 抵押保底阈件（鸡生蛋死锁根修——trading 可用被
-- sweep 扫光→adjEq≈0→max-loan≈0→eval 恒拒 borrow_quota_insufficient→
-- 提案永不 armed→无意向登记→sweep 继续扫，死锁环每环实证在案）。
-- reserve=$500：probe$50 级量测足矣；额度随抵押物线性，实测值低估满额
-- 能力（spec 注释钉——判据只钉锚币非尘量级）。exec_enabled=false→键在
-- 而失效不豁免（休眠不留钱——回滚径=闸本身）；键缺=fail-closed 豁免。
INSERT INTO app_config(key,value) VALUES
  ('margin.collateral_reserve_usd','500')
ON CONFLICT (key) DO NOTHING;
