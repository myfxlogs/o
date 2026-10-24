-- 0026 快照表废弃（T-0048 三令终裁：快照=全域禁用物——判定面全改 venue 现读，
-- 历史事实由 md_account_events append-only 行覆盖，两表 DROP 不迁数）。
DROP TABLE IF EXISTS fin_venue_account_snapshot;
DROP TABLE IF EXISTS md_venue_fact;
