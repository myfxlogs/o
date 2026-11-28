-- 0279 fin_valuation_ccy 估值需求币发布表（O-41395 件②——BTC-USDT
-- tick 死订根修：collector 活跃集=动态持仓联动册（legset.ActiveSetUnion
-- 多臂派生），DCD 策略退役后「估值仍需币」无臂可出=订阅真空实证
-- （2026-11-17 BTC-USDT 行冻结）。估值所需币集=FundCcy∪持仓视图 ccy
-- ——fin 域数据 PG-md 不可见（T-0187 域禁引律），按 fin_strategy 先例
-- 走 bridge-reverse：本表=SG 正本，o-core 发布器维护（集合同步语义
-- ——在集 upsert、出集 DELETE），ev_valuation 帧→projector 反向镜像
-- PG-md 只读副本→legset.ValuationCcyArm 入订阅集+写谓词同源。
CREATE TABLE IF NOT EXISTS fin_valuation_ccy (
    ccy        text PRIMARY KEY,
    src        text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE fin_valuation_ccy IS
  '估值需求币发布表（O-41395）——FundCcy∪fin_position_view.ccy 集合同步；projector 反向镜像 PG-md，legset.ValuationCcyArm 订阅集/写谓词同源消费';
COMMENT ON COLUMN fin_valuation_ccy.src IS '需求源标记（fundccy|view|fundccy+view——出集 DELETE 行级同步）';

-- 事件脊触发器（fin_0028 ev_notify 同范式——行变 NOTIFY 唤 projector
-- 反向 diff；FOR EACH STATEMENT 语句级，发布器同值跳写防空醒）。
CREATE OR REPLACE TRIGGER ev_valuation_trg AFTER INSERT OR UPDATE OR DELETE ON fin_valuation_ccy
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_valuation');

-- 桥域授权钉（check_bridge_registry.sh §① 钦定矩阵——bridge-reverse 源表
-- projector_rw SELECT 必在+research_rw SELECT 按需读；角色缺席测试库
-- 跳过不炸，fin_0173/fin_0256 同径）。
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='projector_rw') THEN
        EXECUTE 'GRANT SELECT ON fin_valuation_ccy TO projector_rw';
    END IF;
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='research_rw') THEN
        EXECUTE 'GRANT SELECT ON fin_valuation_ccy TO research_rw';
    END IF;
END $$;

-- 保留账入册（T-0153 清道夫 fail-closed 同 fin_active_set 先例——
-- 状态表不删行，键域封顶=估值币数百级内）。
INSERT INTO fin_data_retention(table_name,class,note)
VALUES ('fin_valuation_ccy','state','状态表不删行——估值需求币发布面（projector 反向镜像源）')
ON CONFLICT (table_name) DO NOTHING;
