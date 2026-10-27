-- fin_0074 账本口径三类分账（T-0185 业主令：移动和校准不算盈利）——
-- kind→class 钦定单源册+fin_cashflow.class 列回填+trigger 新写即载。
CREATE TABLE IF NOT EXISTS fin_cashflow_kind_class (
    kind  text PRIMARY KEY,
    class text NOT NULL CHECK (class IN ('income','transfer','adjustment')),
    note  text NOT NULL DEFAULT ''
);
INSERT INTO fin_cashflow_kind_class(kind,class,note) VALUES
    ('interest','income','活期利息——真赚'),
    ('funding_settle','income','资金费结算——真赚'),
    ('trade_pnl','income','价差真赚'),
    ('fee','income','交易费负扣——真亏计亏'),
    ('spot_buy','income','现货买腿（平仓仓内=置换进出，消费侧细判）'),
    ('spot_sell','income','现货卖腿（同 spot_buy 双侧）'),
    ('earn_in','transfer','Earn 回款——自有资金内部移动非赚'),
    ('earn_out','transfer','Earn 申购扣款——同移动'),
    ('deposit','transfer','入金注资——非盈利'),
    ('withdraw','transfer','撤资——非亏损'),
    ('recon_adjust','adjustment','对账校准——账面掰向 venue 真实的修正项'),
    ('dust_in','transfer','残币归口收款——内部移动'),
    ('dust_out','transfer','残币归口付款——同移动'),
    ('supply_income','income','理财收益——真赚'),
    ('airdrop','income','空投/奖励——真赚')
ON CONFLICT (kind) DO NOTHING;

ALTER TABLE fin_cashflow ADD COLUMN IF NOT EXISTS class text NOT NULL DEFAULT '';

-- 既有行回填（映射刷一次——未知 kind=记册告警行不落 class 空判=
-- 计入口径侧 fail-closed 手查）。
UPDATE fin_cashflow c SET class=m.class
FROM fin_cashflow_kind_class m WHERE c.kind=m.kind;

-- 新写即载（trigger——单源册每次写时解 class；未知 kind 留空+拦不住写
-- ——写侧 Post 已 kind 闭集校验，册外 kind=维护债非静默）。
CREATE OR REPLACE FUNCTION fin_cashflow_class_stamp() RETURNS trigger AS $$
BEGIN
    SELECT class INTO NEW.class FROM fin_cashflow_kind_class WHERE kind=NEW.kind;
    RETURN NEW;
END $$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS fin_cashflow_class_stamp ON fin_cashflow;
CREATE TRIGGER fin_cashflow_class_stamp
    BEFORE INSERT OR UPDATE OF kind ON fin_cashflow
    FOR EACH ROW EXECUTE FUNCTION fin_cashflow_class_stamp();

INSERT INTO fin_data_retention(table_name,class,note)
VALUES('fin_cashflow_kind_class','permanent','账本口径语义册——永久')
ON CONFLICT (table_name) DO UPDATE SET class='permanent';
