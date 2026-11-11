-- 0163 ops_event 探测查询索引（2026-11-10 SG CPU 烧实证——stuck_esc 自连探
-- 走 pkey id>范围扫 8k 行×每 e1,单后端 100% 单核 2min+/拍,每拍复活;
-- 两枚索引 2026-11-10 已先手上产线止血,此迁移固化形制）。
-- (esc_scope,id): 同 scope 复燃/销案查询定点扫（stuck_esc/escEdgeUV 指纹记忆）
-- (event,id):     处置/升件按事件+id 范围扫（销案行存在性 EXISTS 半连）
CREATE INDEX IF NOT EXISTS ops_event_scope_id ON ops_event (esc_scope, id);
CREATE INDEX IF NOT EXISTS ops_event_event_id ON ops_event (event, id);
