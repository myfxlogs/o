-- 0144 T-0461 simvenue 全周期演习矩阵——fin_exec_drill 增 cell_key 可空列。
--   格凭据=模板×venue对×方向#面值档（机读键，格式=excell.CellKey 唯一真源）；
--   毕业闸径不变（latest verdict by strategy_ref——cell_key=NULL 旧行语义零触），
--   部署前置闸径=armed→open 消费 latest verdict per cell_key（缺/败 fail-closed）。
--   ephemeral 仿真策略行 24h 保留期自清（同 T-0426 语义——凭据留档审计）。
ALTER TABLE fin_exec_drill ADD COLUMN IF NOT EXISTS cell_key text;
CREATE INDEX IF NOT EXISTS idx_fin_exec_drill_cell ON fin_exec_drill(cell_key, ts DESC)
    WHERE cell_key IS NOT NULL;
COMMENT ON COLUMN fin_exec_drill.cell_key IS '演习矩阵格键（T-0461——template@venuepair/direction#usd=<面值>；NULL=单策略演习旧行）';
