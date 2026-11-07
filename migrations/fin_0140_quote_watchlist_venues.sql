-- 0140 T-0442 件③ 自选报价条 venue 源约束列（业主令 2026-11-07——「Nox项目
-- 中自选报价条编辑可以选择订阅提供报价的broker/cex，引进过来」）：成员行扩
-- venues text[]（裸 venue 值集——空=全部源=现语义零破坏）；报价行取价与订阅
-- 晋升按所选源过滤。Nox venueKey=venue:venueType 双维，本仓 md_ticks 只有
-- venue 单维（fin_0003 建表无 inst_type）——单维落地，候选源=md_ticks 实采
-- ∪md_tick_scan 实扫派生零硬编码词表（0440 通用性律：新 venue 接入零改本面）。
-- 裸值禁冒号 D-496 同规（md_ticks.venue CHECK 同式——数组字面投影检：
-- venues::text 渲染 {a,b} 括号逗号无冒号，元素携冒号即现形必炸）。
ALTER TABLE fin_quote_watchlist ADD COLUMN IF NOT EXISTS venues text[] NOT NULL DEFAULT '{}';
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint
      WHERE conname='fin_quote_watchlist_venues_bare') THEN
    ALTER TABLE fin_quote_watchlist ADD CONSTRAINT fin_quote_watchlist_venues_bare
      CHECK (venues::text NOT LIKE '%:%');
  END IF;
END $$;
COMMENT ON COLUMN fin_quote_watchlist.venues IS
  '源约束集（T-0442——裸 venue 值禁冒号；空=全部源=最新跨源）';
