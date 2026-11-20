-- 0217 o-core 换件部署窗配置键（工单#25760——deploy_watch 谱系判命中
-- cmd/core 依赖闭包=镜像必换+钱径重启：随 merge 即换废止，收窄到 UTC
-- 低活动窗集中放件；窗外入窗队列让位，越窗未放升件）。
-- 取值理由（票面 2026-11-20 已载 ops_event#25794）：19:00-21:00 UTC——
-- fin_execution 90d 按 UTC 时 19-21 仅 4/1232（0.3%），13-17 点 1081（88%）；
-- ops_event 14d 19-21=645-790/日峰 2318（02 点），属 18-23 点低频带；
-- 2h 窗=15min 拍约 8 次重试余量。max_hold=26h=24h 日窗+2h 跨度+漏放一次
-- 可检出面。键类=dm（系统自裁部署时窗——分册 scripts/config-write-domain.tsv
-- 同集镜像 dmwrite.go；DM 改值走 DMSet 三拍留参数调整行）。
INSERT INTO app_config(key,value) VALUES
  ('deploy.ocore_window_utc','"19:00-21:00"'),
  ('deploy.ocore_window_max_hold_h','"26"')
ON CONFLICT (key) DO NOTHING;
