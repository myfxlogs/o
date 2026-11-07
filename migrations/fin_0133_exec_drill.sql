-- 0133 T-0426 执行径演习凭据表（纸面跑 venue 语义全径——毕业闸判据件）。
--   业主令 2026-11-07：「每策略毕业前就该检查订单会不会出错」——probe→live
--   判据增「执行径演习通过」：该策略型的开→平→对账全径仿真跑绿才准 live。
--   判据件缺/演习败=毕业闸 fail-closed（同经济判据待遇——lifecycle.promote
--   probe→live 循环读本表 latest 行，verdict='pass' 缺位不晋）。
--   verdict 语义：pass=开→平→对账全径绿；fail=任一环账实不平/drift 假报/
--   残袋非零/拒单吞码（detail 携败因——升件行同拍落 ops_event）。
--   rules_version=引擎指纹（RULES_VERSION 族）——「哪套规则判的」审计锚。
CREATE TABLE IF NOT EXISTS fin_exec_drill (
    id            bigserial PRIMARY KEY,
    strategy_ref  bigint NOT NULL,
    inst_id       text NOT NULL,
    verdict       text NOT NULL CHECK (verdict IN ('pass','fail')),
    rules_version text NOT NULL DEFAULT '',
    detail        text NOT NULL DEFAULT '',
    ts            timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_fin_exec_drill_strategy ON fin_exec_drill(strategy_ref, ts DESC);
COMMENT ON TABLE fin_exec_drill IS '执行径演习凭据（T-0426——probe→live 毕业闸判据件：latest verdict=pass 才准 live，缺/败 fail-closed）';
COMMENT ON COLUMN fin_exec_drill.rules_version IS '仿真引擎指纹（RULES_VERSION 族——演习时规则集版本，审计可验）';
