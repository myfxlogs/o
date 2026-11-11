-- 0164 fin_condition_gap pending 事件源（T-0511 飞书交互端——D-0008-B
-- 禁周期轮询：pg_notify 表级事件=确定性推送，桥件 LISTEN 即醒；
-- payload=请求 id——notify 不载正文只敲门，读仍走表正源）。
CREATE OR REPLACE FUNCTION fin_condition_gap_notify() RETURNS trigger AS $$
BEGIN
  PERFORM pg_notify('condreq_pending', NEW.id::text);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS fin_condition_gap_pending_notify ON fin_condition_gap;
CREATE TRIGGER fin_condition_gap_pending_notify
  AFTER INSERT OR UPDATE OF state ON fin_condition_gap
  FOR EACH ROW WHEN (NEW.state = 'pending')
  EXECUTE FUNCTION fin_condition_gap_notify();
