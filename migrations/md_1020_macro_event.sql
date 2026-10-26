-- 0020 高冲击日历事实表（T-0029：种成事实最省——零第三方 API 依赖；
-- v1 仅评估卡 evidence 注解非闸不改判据）。日期以官方公布历录种子。
-- T-0167 分链：md_macro_event=第八桥表正本（PG-md——D-0032 §二裁，report 留 SG
-- 读镜像 fin_0067，research 读 PG-md 本地）；research.macro.horizon_days 配置键在 fin 链 fin_0020。
CREATE TABLE IF NOT EXISTS md_macro_event (
    event_date date PRIMARY KEY,
    kind       text NOT NULL CHECK (kind IN ('fomc','cpi','nfp')),
    title      text NOT NULL,
    source     text NOT NULL DEFAULT 'seed'
);
COMMENT ON TABLE md_macro_event IS '宏观高冲击日历事实域（md_ 前缀——种成事实非行情流）';

-- 种子：2026 余下 FOMC（官方历）+ CPI/NFP 月度例行（美国劳工部公布历——
-- 例行日=按月首个周三/周五口径近似录入，种子域注 source='seed'）。
INSERT INTO md_macro_event(event_date,kind,title) VALUES
  ('2026-10-28','fomc','FOMC 利率决议'),
  ('2026-12-09','fomc','FOMC 利率决议'),
  ('2027-01-27','fomc','FOMC 利率决议'),
  ('2027-03-17','fomc','FOMC 利率决议'),
  ('2027-04-28','fomc','FOMC 利率决议'),
  ('2027-06-16','fomc','FOMC 利率决议'),
  ('2027-07-28','fomc','FOMC 利率决议'),
  ('2027-09-15','fomc','FOMC 利率决议'),
  ('2026-11-12','cpi','CPI 月度发布'),
  ('2026-12-10','cpi','CPI 月度发布'),
  ('2027-01-13','cpi','CPI 月度发布'),
  ('2027-02-10','cpi','CPI 月度发布'),
  ('2027-03-11','cpi','CPI 月度发布'),
  ('2027-04-13','cpi','CPI 月度发布'),
  ('2027-05-12','cpi','CPI 月度发布'),
  ('2027-06-10','cpi','CPI 月度发布'),
  ('2027-07-14','cpi','CPI 月度发布'),
  ('2027-08-12','cpi','CPI 月度发布'),
  ('2027-09-11','cpi','CPI 月度发布'),
  ('2027-10-13','cpi','CPI 月度发布'),
  ('2026-11-06','nfp','NFP 非农就业月度'),
  ('2026-12-04','nfp','NFP 非农就业月度'),
  ('2027-01-08','nfp','NFP 非农就业月度'),
  ('2027-02-05','nfp','NFP 非农就业月度'),
  ('2027-03-05','nfp','NFP 非农就业月度'),
  ('2027-04-02','nfp','NFP 非农就业月度'),
  ('2027-05-07','nfp','NFP 非农就业月度'),
  ('2027-06-04','nfp','NFP 非农就业月度'),
  ('2027-07-02','nfp','NFP 非农就业月度'),
  ('2027-08-06','nfp','NFP 非农就业月度'),
  ('2027-09-03','nfp','NFP 非农就业月度'),
  ('2027-10-01','nfp','NFP 非农就业月度')
ON CONFLICT (event_date) DO NOTHING;
