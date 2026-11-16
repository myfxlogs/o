-- 1096 perp-trend-daily 方向趋势族 md 侧（工单#11481：假想卡#172286
-- okx/perp-trend-daily parked 解锁径骨架件——评估数学=卡薄口径落地式，
-- 全参数 app_config 机读禁虚构常数；判据键在 fin 链 fin_0195 同构）：
-- fin_opp_template 登记 perp-trend-daily（模板册正本在 md——fin_0052 分链；
-- params=腿原语+信号参数快照，判据真值在 fin 链 app_config）。
-- 纸面域裁定同棒1：perp-trend-daily=paper-only（真单径另棒——execfam
-- .PaperOnly 册载，excell 免演登记+Drillable=false 拦 probe→live）。
-- 假想卡 adopt=席权 decide 窄门另件，本迁移不写 hypothesis。

INSERT INTO fin_opp_template(template,venue,surface,params) VALUES
  ('perp-trend-daily','okx','perp-trend-daily',
   '{"legs":["perp_long","perp_short"],"signal":"donchian_breakout","settle":"mtm"}')
ON CONFLICT DO NOTHING;
