-- 1084 T-0310 返修①：md_ctx_fact 幂等去重键（source,kind,dedupe_key——
-- 重复拍=同键拒重行；NULL 键不参与唯一约束留口）。
ALTER TABLE md_ctx_fact ADD COLUMN IF NOT EXISTS dedupe_key text;
CREATE UNIQUE INDEX IF NOT EXISTS md_ctx_fact_dedupe_uq
  ON md_ctx_fact(source,kind,dedupe_key) WHERE dedupe_key IS NOT NULL;
