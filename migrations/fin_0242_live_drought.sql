-- 0242 live 终局断流探针键+P0 族谱惧值固化（工单O-35729——业主令
-- 2026-11-23②③④：钱躺着=P0 顶格，闲置族权重不漂）：
-- ① dm.hunger.live_drought_days——live 终局断流窗（初值 3d 对齐
--   zero_proposal/open_drought_days 票面钦例；探针 fail-closed 键缺不探，
--   落键防哑针是本件验收项）；
-- ② ops_esc_valence upsert——live_drought 新类 100 顶格（从未毕业=P0，
--   超账链断/唤知死 95）；闲置族权重钉死：idle_capital 55→100（钱躺着
--   本尊）、live_dud 75→95、commercial_stall 75→90、opp_eval_stall
--   55→85（scope 级键压过 class 级——ops_esc_pending COALESCE 序在案）；
--   世模/calib_drift 不动（非闲置族——业主令表外）。

INSERT INTO app_config(key,value,updated_by) VALUES
  ('dm.hunger.live_drought_days', '3', 'fin_0242')
ON CONFLICT (key) DO NOTHING;

INSERT INTO ops_esc_valence(esc_key,weight,note) VALUES
  ('live_drought',100,'终局段——从未毕业=P0（业主令2026-11-23②钱闲置顶格）'),
  ('资金闸/idle_capital',100,'钱躺着本尊——闲置=系统病非排前好事（业主令③）'),
  ('商业产出/live_dud',95,'live层哑弹相邻族'),
  ('商业产出/commercial_stall',90,'商业产出断档'),
  ('数据缺/opp_eval_stall',85,'评估流断档')
ON CONFLICT (esc_key) DO UPDATE SET weight=EXCLUDED.weight, note=EXCLUDED.note;
