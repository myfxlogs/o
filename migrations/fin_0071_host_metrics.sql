-- fin_0071 主机资源面（T-0173 业主令——面板加 CPU/内存/磁盘/网络服务器状态）。
-- fin_host_metrics=每主机一行状态表（D-0018 有界态语义：PK(host) upsert，
-- 30s 拍覆写零增长）；写方=各机进程自报（SG core 写 sg / dev collector 经
-- ROSTER 池写 dev——与席位 beat 同频同径零新 env）；读方=GetHostMetrics。
-- 陈阈 2min（超=采集断，面板 stale 显形不假值）。
CREATE TABLE IF NOT EXISTS fin_host_metrics (
    host          text PRIMARY KEY,
    ts            timestamptz NOT NULL DEFAULT now(),
    cores         int  NOT NULL,
    cpu_pct       real NOT NULL,
    mem_total_mb  bigint NOT NULL,
    mem_used_mb   bigint NOT NULL,
    disk_total_gb real NOT NULL,
    disk_used_gb  real NOT NULL,
    net_rx_kbps   real NOT NULL,
    net_tx_kbps   real NOT NULL,
    uptime_s      bigint NOT NULL
);
COMMENT ON TABLE fin_host_metrics IS
  '主机资源状态表（每机一行 upsert——面板服务器状态瓦片供数；采集断=行陈非删）';

-- 写方角色授权（collector_rw=dev 采集自报径；SG core 写方=表主 o 免授）。
GRANT SELECT, INSERT, UPDATE ON fin_host_metrics TO collector_rw;
