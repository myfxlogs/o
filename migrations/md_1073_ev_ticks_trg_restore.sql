-- md_1073 ev_ticks_trg 回填（T-0194——md_1049 DROP TABLE md_ticks 重建
-- 未回填：dev 侧投影臂 ev_ticks 通道死源）。
CREATE OR REPLACE TRIGGER ev_ticks_trg AFTER INSERT ON md_ticks
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_ticks');
