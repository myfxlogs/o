-- fin_0073 app_config 审计列（T-0178：live_switch 翻位者无审计痕病灶——
-- 谁翻的何时翻的不可考；写径带写者身份，既有行回填 'legacy'）。
ALTER TABLE app_config ADD COLUMN IF NOT EXISTS updated_by text NOT NULL DEFAULT '';
UPDATE app_config SET updated_by='legacy' WHERE updated_by='';
