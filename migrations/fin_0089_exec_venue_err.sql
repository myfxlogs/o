-- fin_0089 fin_execution venue 拒因列（T-0259：dcdOpen trade 步 venue 拒码+拒文
-- 持久化——o-core 容器重建毁 stdout 实证，拒因落库不毁证）。同构位=fin_order_leg
-- .venue_err_code/venue_err_msg（0009 先例）上移执行行层；quote 步上游拦无 exec
-- 行落 fin_eval_reject（reason=dcd_quote_reject:<code>，本件不涉表构）。
ALTER TABLE fin_execution ADD COLUMN IF NOT EXISTS venue_err_code text;
ALTER TABLE fin_execution ADD COLUMN IF NOT EXISTS venue_err_msg  text;
COMMENT ON COLUMN fin_execution.venue_err_code IS
  'venue 原始拒码（内层 sCode 优先、envelope code 兜底——原始值非归一化枚举；非 venue 拒=NULL）';
COMMENT ON COLUMN fin_execution.venue_err_msg IS
  '败因原文（venue 拒文或本端错误串——传输/解析败亦留痕不吞）';
