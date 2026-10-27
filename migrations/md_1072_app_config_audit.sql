-- md_1072 app_config 审计列副本同形（T-0178：桥表两链同形闸——
-- fin_0073 同 commit 落；副本侧 updated_by 随正本投影同步）。
ALTER TABLE app_config ADD COLUMN IF NOT EXISTS updated_by text NOT NULL DEFAULT '';
