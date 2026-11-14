-- fin_0183 在办检查点事件脊（在办契读侧件——「每场各自在办什么」前端秒推）。
-- ops_event 检查点生命周期行落库即 pg_notify('ev_headless')——搭 ev_headless
-- 现成 LISTEN 频道（T-0048 事驱脊律；前端 Watch 消费 ev_headless 已会重取，
-- 前端零改）。行级 WHEN 收窄：只在办/收口/果/败 + 处置引「在办#」销账行——
-- ops_event 高写面全量触发=notify 洪泛（fin_0064 同病灶律），只推检查点族
-- （唤起/叙事入族——唤知生命周期帧即在办面上下文，sys_push 活卡同脊刷新）。
CREATE OR REPLACE TRIGGER ev_doing_trg
  AFTER INSERT ON ops_event
  FOR EACH ROW
  WHEN (NEW.event IN ('在办','会话收口','唤起','唤起果','唤起败','叙事')
        OR (NEW.event = '处置' AND NEW.note LIKE '%在办#%'))
  EXECUTE FUNCTION ev_notify('ev_headless');

COMMENT ON TRIGGER ev_doing_trg ON ops_event IS
  '在办检查点族→ev_headless 一跳(在办契读侧件——行级WHEN收窄防notify洪泛)';
