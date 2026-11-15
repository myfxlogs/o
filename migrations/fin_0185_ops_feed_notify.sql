-- fin_0185 动作流水事件脊（业主令 2026-11-15④：O 每个动作更新都发系统频
-- =「我在做什么」消息流——在办活卡是此刻快照，动作流水是逐条时间线）。
-- ops_event 动作行落库即 pg_notify('ev_feed')→o-core feed_push LISTEN 即醒。
-- 行级 WHEN 收窄防 notify 洪泛（fin_0064 同病灶律），剔：
--   量表=心跳帧分钟级噪音（自模账/盈利报告走 relay digest 周期报）；
--   唤起/唤起果/唤起败=会话生命周期管道音（收口行携果即在流内，活卡同显）；
--   条件请求=交互卡专径（condreq_pending 一跳不抄送）；
--   求助值守=relay urgent 道专径。
CREATE OR REPLACE TRIGGER ev_feed_trg
  AFTER INSERT ON ops_event
  FOR EACH ROW
  WHEN (NEW.event NOT IN ('量表','唤起','唤起果','唤起败','条件请求','求助值守'))
  EXECUTE FUNCTION ev_notify('ev_feed');

COMMENT ON TRIGGER ev_feed_trg ON ops_event IS
  '动作流水→ev_feed一跳(系统频feed件——行级WHEN剔心跳/生命周期/专径族)';
