-- fin_0074 主机资源事件帧（T-0173 业主令：主机资源做成流——30s 自报拍
-- 逐拍 NOTIFY ev_hostmetrics，Watch 脊推送→面板随帧重抓；事件驱动律
-- D-0008——前端零轮询，拍频 2 主机×30s=4 帧/分可控不洪泛）。
CREATE OR REPLACE TRIGGER ev_hostmetrics_trg
AFTER INSERT OR UPDATE ON fin_host_metrics
FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_hostmetrics');
