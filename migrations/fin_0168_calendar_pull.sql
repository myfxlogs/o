-- 0168 日历拉取活性账（工单#10693/升件#10654——CALENDAR-FEED 探针口径修：
-- 拉环逐拍实拉落行（ok/err/事件数），探针改读拉取活性锚非推件产出——
-- 零 high 冲击窗=设计缺席不误判；拉环死=行龄断真病灶（读败档仍管上游拒因）。
CREATE TABLE IF NOT EXISTS md_calendar_pull (
    id      bigserial PRIMARY KEY,
    ts      timestamptz NOT NULL DEFAULT now(),
    ok      bool NOT NULL,               -- 全链通（200+decode+非空）=true;fetch/HTTP/decode/空返=false
    events  int NOT NULL DEFAULT 0,      -- 当拍 XML 事件行数（ok 时）
    err     text NOT NULL DEFAULT ''     -- 败因相（fetch/http_<code>/decode/empty）
);
COMMENT ON TABLE md_calendar_pull IS
  '日历拉取活性账（工单#10693）——calendar 环逐拍实拉落行；CALENDAR-FEED 探针活性锚改读本表，零 high 窗不判死';
