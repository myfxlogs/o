-- fin_0077 ev_account_trg 回填（T-0190——fin_0049 DROP+CREATE md_account_events
-- 吞触发器=事件脊断肋：私有帧 INSERT 零 NOTIFY→ledger scan/holding 欠驱动
-- →restOKAt 间隙假死。同族病灶防：DROP+CREATE 迁移吞零件须清单对拍）。
CREATE OR REPLACE TRIGGER ev_account_trg AFTER INSERT ON md_account_events
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_account');
