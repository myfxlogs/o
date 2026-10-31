-- 0093 T-0277：ops_headless.state 五态→六态列注同步（exit 正常终态新增——
--   witness rc=0 退出且无收口证=已退，dead 收窄=rc≠0 见证或无锚消失；
--   数据列 text 零 schema 变更，仅列注随判态语义更新）。
COMMENT ON COLUMN ops_headless.state IS
  '六态机键：alive在飞|stalled停滞(日志静默>30min进程在)|dead死(rc≠0见证退出/无锚消失)|overdue逾期(在飞>8h)|done完工(收口铁证)|exit已退(rc=0见证退出无收口——正常收工非异常)——服务端判态前端零判据';
