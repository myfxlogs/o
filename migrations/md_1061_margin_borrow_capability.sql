-- 1061 margin-borrow 能力面登记（T-0216 条件缺口件——quarter_basis_short
-- spot_short 腿=保证金借币卖：endpoint 在册 enabled=false，读面实测启用径
-- 由决策席批（borrow-repay=写面钱径，入册≠启用；history=读面先验方言对拍）。
INSERT INTO fin_capability_surface(venue,surface,endpoint_family,product_family,rw_class,cost_attrs,enabled) VALUES
('okx','margin-borrow-repay','/api/v5/account/borrow-repay','borrow','write','{"idem":"none","mode":"borrow|repay"}',false),
('okx','margin-borrow-history','/api/v5/account/borrow-repay-history','borrow','read','{"src":"rest"}',false)
ON CONFLICT (venue,surface) DO NOTHING;
