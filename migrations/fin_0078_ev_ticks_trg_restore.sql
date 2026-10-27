-- fin_0078 ev_ticks_trg 回填（T-0194——fin_0049 DROP+CREATE 吞触发器第二件
-- casualty：SG 侧 md_ticks 镜像表 UPSERT/投影批写零 NOTIFY=仓巡哨聋于
-- 价格事件；与 ev_account_trg（fin_0077）同族根修）。
CREATE OR REPLACE TRIGGER ev_ticks_trg AFTER INSERT ON md_ticks
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_ticks');
