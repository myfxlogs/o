-- 0146 T-0464 反问环——单仓盈亏归因三分落位+问链立案写径授权。
-- ① fin_position.pnl_attribution——每笔平仓判词「为什么亏/为什么赚」归因
--   四值闭集（market/code/ops/unattributed；机判序 code锚→ops锚→market锚→
--   零锚=unattributed 显式判词非沉默——业主令钦）。postmortem 结案判定机
--   拍内同槽落位（trigger_verdict「怎么坏」轴并存——两轴不混）；人裁可改判，
--   机判原值留 ops_decision outcome payload/监察案 numbers 审计链，改判
--   留痕走裁定席既有 ops_decision/ops_event 道。
-- ② roster_rw 补 fin_econ_lesson 列级 INSERT——问链扫描立案写径
--   （scripts/meta/why_chain_scan.sh→verdict_tick.sh file）经 FIN_PG_DSN
--   写 rootcause_diag 案卷；现状 roster_rw 仅 SELECT,UPDATE（fin_0103
--   裁定席消费面）INSERT 零授权实勘。列级授 kind/entity_ref/numbers——
--   status/verdict_ref 走 DEFAULT（pending）不可伪造（fin_0139 列级判例
--   同径）；UPDATE/DELETE 仍零授。
-- ③ o_fin_ro 补 fin_order_leg SELECT——问链 fill 段错因取证读面
--   （why_chain_scan 段取数走 FIN_PG_RO_DSN 窄读册；fin_order_leg 系
--   o_fin_ro 白名单最后缺口——fin_0109 fin_execution 同径先例）。
ALTER TABLE fin_position ADD COLUMN IF NOT EXISTS pnl_attribution text;
COMMENT ON COLUMN fin_position.pnl_attribution IS '单仓盈亏归因判词（T-0464 四值闭集 market/code/ops/unattributed——机判序 code锚→ops锚→market锚→零锚unattributed；postmortem 结案判定机写，人裁可改判留裁定道痕）';

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'roster_rw') THEN
    EXECUTE 'GRANT INSERT (kind,entity_ref,numbers) ON fin_econ_lesson TO roster_rw';
    EXECUTE 'GRANT USAGE ON SEQUENCE fin_econ_lesson_id_seq TO roster_rw';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'o_fin_ro') THEN
    EXECUTE 'GRANT SELECT ON fin_order_leg TO o_fin_ro';
  END IF;
END $$;
